package goetty

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// AppOption application option
type AppOption[IN any, OUT any] func(*server[IN, OUT])

// WithAppHandleSessionFunc set the app handle session func
func WithAppHandleSessionFunc[IN any, OUT any](value func(IOSession[IN, OUT]) error) AppOption[IN, OUT] {
	return func(s *server[IN, OUT]) {
		s.options.handleSessionFunc = value
	}
}

// WithAppLogger set logger for application
func WithAppLogger[IN any, OUT any](logger *zap.Logger) AppOption[IN, OUT] {
	return func(s *server[IN, OUT]) {
		s.logger = logger
	}
}

// WithAppSessionBucketSize set the number of maps to store session
func WithAppSessionBucketSize[IN any, OUT any](value uint64) AppOption[IN, OUT] {
	return func(s *server[IN, OUT]) {
		s.options.sessionBucketSize = value
	}
}

// WithAppSessionBucketSize set the app session aware
func WithAppSessionAware[IN any, OUT any](value IOSessionAware[IN, OUT]) AppOption[IN, OUT] {
	return func(s *server[IN, OUT]) {
		s.options.aware = value
	}
}

// WithAppSessionOptions set options to create new connection
func WithAppSessionOptions[IN any, OUT any](options ...Option[IN, OUT]) AppOption[IN, OUT] {
	return func(s *server[IN, OUT]) {
		s.options.sessionOpts = options
	}
}

// WithAppTLS set tls config for application
func WithAppTLS[IN any, OUT any](tlsCfg *tls.Config) AppOption[IN, OUT] {
	return func(s *server[IN, OUT]) {
		for idx, listener := range s.listeners {
			s.listeners[idx] = tls.NewListener(listener, tlsCfg)
		}
	}
}

// WithAppTLSFromKeys set tls config from cert and key files for application
func WithAppTLSFromCertAndKey[IN any, OUT any](
	certFile string,
	keyFile string,
	caFile string,
	insecureSkipVerify bool) AppOption[IN, OUT] {
	return func(s *server[IN, OUT]) {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			panic(err)
		}
		var caPool *x509.CertPool
		if caFile != "" {
			certBytes, err := os.ReadFile(caFile)
			if err != nil {
				panic(err)
			}
			caPool = x509.NewCertPool()
			ok := caPool.AppendCertsFromPEM(certBytes)
			if !ok {
				panic("failed to parse root certificate")
			}
		}

		for idx, listener := range s.listeners {
			s.listeners[idx] = tls.NewListener(listener, &tls.Config{
				Certificates:       []tls.Certificate{cert},
				InsecureSkipVerify: insecureSkipVerify,
				ClientAuth:         tls.RequireAndVerifyClientCert,
				ClientCAs:          caPool,
			})
		}
	}
}

// NetApplication is a network based application
type NetApplication[IN any, OUT any] interface {
	// Start start the transport server
	Start() error
	// Stop stop the transport server
	Stop() error
	// GetSession get session
	GetSession(uint64) (IOSession[IN, OUT], error)
}

type sessionMap[IN any, OUT any] struct {
	sync.RWMutex
	sessions map[uint64]IOSession[IN, OUT]
}

type server[IN any, OUT any] struct {
	logger     *zap.Logger
	listeners  []net.Listener
	wg         sync.WaitGroup
	sessions   map[uint64]*sessionMap[IN, OUT]
	handleFunc func(IOSession[IN, OUT], IN, uint64) error

	mu struct {
		sync.RWMutex
		running bool
	}

	atomic struct {
		id uint64
	}

	options struct {
		sessionOpts       []Option[IN, OUT]
		sessionBucketSize uint64
		aware             IOSessionAware[IN, OUT]
		handleSessionFunc func(IOSession[IN, OUT]) error
	}
}

// NewApplicationWithListener returns a net application with listener
func NewApplicationWithListeners[IN any, OUT any](
	listeners []net.Listener,
	handleFunc func(IOSession[IN, OUT], IN, uint64) error,
	opts ...AppOption[IN, OUT]) (NetApplication[IN, OUT], error) {
	s := &server[IN, OUT]{
		listeners:  listeners,
		handleFunc: handleFunc,
	}

	for _, opt := range opts {
		opt(s)
	}

	s.adjust()

	addresses := "["
	for idx, listener := range s.listeners {
		addresses += listener.Addr().String()
		if idx != len(s.listeners)-1 {
			addresses += ","
		}
	}
	addresses += "]"
	s.logger = s.logger.With(zap.String("listen-addresses", addresses))
	s.sessions = make(map[uint64]*sessionMap[IN, OUT], s.options.sessionBucketSize)
	for i := uint64(0); i < s.options.sessionBucketSize; i++ {
		s.sessions[i] = &sessionMap[IN, OUT]{
			sessions: make(map[uint64]IOSession[IN, OUT]),
		}
	}
	return s, nil
}

// NewApplication returns a application
func NewApplication[IN any, OUT any](
	address string,
	handleFunc func(IOSession[IN, OUT], IN, uint64) error,
	opts ...AppOption[IN, OUT]) (NetApplication[IN, OUT], error) {
	network, address, err := parseAddress(address)
	if err != nil {
		return nil, err
	}

	listenConfig := &net.ListenConfig{
		Control: listenControl,
	}

	listener, err := listenConfig.Listen(context.TODO(), network, address)
	if err != nil {
		return nil, err
	}

	return NewApplicationWithListeners([]net.Listener{listener}, handleFunc, opts...)
}

// NewApplicationWithListenAddress create a net application with listen multi addresses
func NewApplicationWithListenAddress[IN any, OUT any](
	addresses []string,
	handleFunc func(IOSession[IN, OUT], IN, uint64) error,
	opts ...AppOption[IN, OUT]) (NetApplication[IN, OUT], error) {
	listeners := make([]net.Listener, 0, len(addresses))
	for _, address := range addresses {
		network, address, err := parseAddress(address)
		if err != nil {
			return nil, err
		}

		listenConfig := &net.ListenConfig{
			Control: listenControl,
		}

		listener, err := listenConfig.Listen(context.TODO(), network, address)
		if err != nil {
			return nil, err
		}
		listeners = append(listeners, listener)
	}

	return NewApplicationWithListeners(listeners, handleFunc, opts...)
}

func (s *server[IN, OUT]) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.mu.running {
		return nil
	}

	s.mu.running = true
	s.doStart()
	s.logger.Debug("application started")
	return nil
}

func (s *server[IN, OUT]) Stop() error {
	s.mu.Lock()
	if !s.mu.running {
		s.mu.Unlock()
		return nil
	}
	s.mu.running = false
	s.mu.Unlock()

	var errors []error
	for _, listener := range s.listeners {
		if err := listener.Close(); err != nil {
			errors = append(errors, err)
		}
	}
	if len(errors) > 0 {
		return errors[0]
	}

	s.logger.Debug("application listener closed")
	s.wg.Wait()

	// now no new connection will added, close all active sessions
	for _, m := range s.sessions {
		m.Lock()
		for k, rs := range m.sessions {
			delete(m.sessions, k)
			if err := rs.Disconnect(); err != nil {
				s.logger.Error("session closed failed",
					zap.Error(err))
			}
		}
		m.Unlock()
	}
	s.logger.Debug("application stopped")
	return nil
}

func (s *server[IN, OUT]) GetSession(id uint64) (IOSession[IN, OUT], error) {
	if !s.isStarted() {
		return nil, errors.New("server is not started")
	}

	m := s.sessions[id%s.options.sessionBucketSize]
	m.RLock()
	session := m.sessions[id]
	m.RUnlock()
	return session, nil
}

func (s *server[IN, OUT]) adjust() {
	s.logger = adjustLogger(s.logger)
	s.options.sessionOpts = append(s.options.sessionOpts,
		WithSessionLogger[IN, OUT](s.logger))
	if s.options.sessionBucketSize == 0 {
		s.options.sessionBucketSize = defaultSessionBucketSize
	}
}

func (s *server[IN, OUT]) doStart() {
	s.logger.Debug("application accept loop started")
	defer func() {
		s.logger.Debug("application accept loop stopped")
	}()

	listenFunc := func(listener net.Listener) {
		defer s.wg.Done()

		var tempDelay time.Duration
		for {
			conn, err := listener.Accept()
			if err != nil {
				if !s.isStarted() {
					return
				}

				if ne, ok := err.(net.Error); ok && ne.Timeout() {
					if tempDelay == 0 {
						tempDelay = 5 * time.Millisecond
					} else {
						tempDelay *= 2
					}
					if max := 1 * time.Second; tempDelay > max {
						tempDelay = max
					}
					time.Sleep(tempDelay)
					continue
				}
				return
			}
			tempDelay = 0

			var options []Option[IN, OUT]
			options = append(options,
				WithSessionConn[IN, OUT](s.nextID(), conn),
				WithSessionLogger[IN, OUT](s.logger),
				WithSessionAware(s.options.aware))
			options = append(options, s.options.sessionOpts...)
			rs := NewIOSession(options...)
			if !s.addSession(rs) {
				if err := rs.Close(); err != nil {
					s.logger.Error("close session failed", zap.Error(err))
				}
				return
			}

			handle := s.options.handleSessionFunc
			if handle == nil {
				handle = s.doConnection
			}
			go func() {
				defer func() {
					if s.deleteSession(rs) {
						if err := rs.Close(); err != nil {
							s.logger.Error("close session failed", zap.Error(err))
						}
					}
				}()
				if err := handle(rs); err != nil {
					s.logger.Error("handle session failed", zap.Error(err))
				}
			}()
		}
	}

	for _, listener := range s.listeners {
		s.wg.Add(1)
		go listenFunc(listener)
	}
}

func (s *server[IN, OUT]) doConnection(rs IOSession[IN, OUT]) error {
	logger := s.logger.With(zap.Uint64("session-id", rs.ID()),
		zap.String("addr", rs.RemoteAddress()))

	logger.Debug("session connected")

	received := uint64(0)
	for {
		msg, err := rs.Read(ReadOptions{})
		if err != nil {
			if err == io.EOF {
				return nil
			}

			logger.Info("session read failed",
				zap.Error(err))
			return err
		}

		received++
		if ce := logger.Check(zap.DebugLevel, "session read message"); ce != nil {
			ce.Write(zap.Uint64("sequence", received))
		}

		err = s.handleFunc(rs, msg, received)
		if err != nil {
			logger.Error("session handle failed, close this session",
				zap.Error(err))
			return err
		}
	}
}

func (s *server[IN, OUT]) addSession(session IOSession[IN, OUT]) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.mu.running {
		return false
	}

	m := s.sessions[session.ID()%s.options.sessionBucketSize]
	m.Lock()
	m.sessions[session.ID()] = session
	m.Unlock()
	return true
}

func (s *server[IN, OUT]) deleteSession(session IOSession[IN, OUT]) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.mu.running {
		return false
	}

	m := s.sessions[session.ID()%s.options.sessionBucketSize]
	m.Lock()
	delete(m.sessions, session.ID())
	m.Unlock()
	return true
}

func (s *server[IN, OUT]) nextID() uint64 {
	return atomic.AddUint64(&s.atomic.id, 1)
}

func (s *server[IN, OUT]) isStarted() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.mu.running
}

func parseAddress(address string) (string, string, error) {
	if !strings.Contains(address, "//") {
		return "tcp4", address, nil
	}

	u, err := url.Parse(address)
	if err != nil {
		return "", "", err
	}

	if strings.ToUpper(u.Scheme) == "UNIX" {
		return u.Scheme, u.Path, nil
	}

	return u.Scheme, u.Host, nil
}
