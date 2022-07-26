package goetty

import (
	"context"
	"errors"
	"io"
	"net"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// AppOption application option
type AppOption func(*server)

// WithAppLogger set logger for application
func WithAppLogger(logger *zap.Logger) AppOption {
	return func(s *server) {
		s.logger = logger
	}
}

// WithAppSessionBucketSize set the number of maps to store session
func WithAppSessionBucketSize(value uint64) AppOption {
	return func(s *server) {
		s.options.sessionBucketSize = value
	}
}

// WithAppSessionBucketSize set the app session aware
func WithAppSessionAware(value IOSessionAware) AppOption {
	return func(s *server) {
		s.options.aware = value
	}
}

// WithAppSessionOptions set options to create new connection
func WithAppSessionOptions(options ...Option) AppOption {
	return func(s *server) {
		s.options.sessionOpts = options
	}
}

// NetApplication is a network based application
type NetApplication interface {
	// Start start the transport server
	Start() error
	// Stop stop the transport server
	Stop() error
	// GetSession get session
	GetSession(uint64) (IOSession, error)
}

type sessionMap struct {
	sync.RWMutex
	sessions map[uint64]IOSession
}

type server struct {
	logger     *zap.Logger
	listener   net.Listener
	closedC    chan struct{}
	sessions   map[uint64]*sessionMap
	handleFunc func(IOSession, any, uint64) error

	mu struct {
		sync.RWMutex
		running bool
	}

	atomic struct {
		id uint64
	}

	options struct {
		sessionOpts       []Option
		sessionBucketSize uint64
		aware             IOSessionAware
	}
}

// NewApplicationWithListener returns a net application with listener
func NewApplicationWithListener(listener net.Listener, handleFunc func(IOSession, any, uint64) error, opts ...AppOption) (NetApplication, error) {
	s := &server{
		listener:   listener,
		handleFunc: handleFunc,
		closedC:    make(chan struct{}),
	}

	for _, opt := range opts {
		opt(s)
	}

	s.adjust()
	s.logger = s.logger.With(zap.String("listen-address", listener.Addr().String()))
	s.sessions = make(map[uint64]*sessionMap, s.options.sessionBucketSize)
	for i := uint64(0); i < s.options.sessionBucketSize; i++ {
		s.sessions[i] = &sessionMap{
			sessions: make(map[uint64]IOSession),
		}
	}
	return s, nil
}

// NewApplication returns a application
func NewApplication(address string, handleFunc func(IOSession, any, uint64) error, opts ...AppOption) (NetApplication, error) {
	network, address, err := parseAdddress(address)
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

	return NewApplicationWithListener(listener, handleFunc, opts...)
}

func (s *server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.mu.running {
		return nil
	}

	s.mu.running = true
	go s.doStart()
	s.logger.Debug("application started")
	return nil
}

func (s *server) Stop() error {
	s.mu.Lock()
	if !s.mu.running {
		s.mu.Unlock()
		return nil
	}
	s.mu.running = false
	s.mu.Unlock()

	if err := s.listener.Close(); err != nil {
		return err
	}
	s.logger.Debug("application listener closed")
	<-s.closedC

	// now no new connection will added, close all active sessions
	for _, m := range s.sessions {
		m.Lock()
		for k, rs := range m.sessions {
			delete(m.sessions, k)
			if err := rs.Close(); err != nil {
				s.logger.Error("session closed failed",
					zap.Error(err))
			}
		}
		m.Unlock()
	}
	s.logger.Debug("application stopped")
	return nil
}

func (s *server) GetSession(id uint64) (IOSession, error) {
	if !s.isStarted() {
		return nil, errors.New("server is not started")
	}

	m := s.sessions[id%s.options.sessionBucketSize]
	m.RLock()
	session := m.sessions[id]
	m.RUnlock()
	return session, nil
}

func (s *server) adjust() {
	s.logger = adjustLogger(s.logger)
	s.options.sessionOpts = append(s.options.sessionOpts, WithSessionLogger(s.logger))
	if s.options.sessionBucketSize == 0 {
		s.options.sessionBucketSize = defaultSessionBucketSize
	}
}

func (s *server) doStart() {
	s.logger.Debug("application accept loop started")
	defer func() {
		s.logger.Debug("application accept loop stopped")
		close(s.closedC)
	}()

	var tempDelay time.Duration
	for {
		conn, err := s.listener.Accept()
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

		var options []Option
		options = append(options,
			WithSessionConn(s.nextID(), conn),
			WithSessionAware(s.options.aware))
		options = append(options, s.options.sessionOpts...)
		rs := NewIOSession(options...)
		if !s.addSession(rs) {
			if err := rs.Close(); err != nil {
				s.logger.Error("close session failed", zap.Error(err))
			}
			return
		}

		go func() {
			defer func() {
				if s.deleteSession(rs) {
					if err := rs.Close(); err != nil {
						s.logger.Error("close session failed", zap.Error(err))
					}
				}
			}()
			s.doConnection(rs)
		}()
	}
}

func (s *server) doConnection(rs IOSession) error {
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
			ce.Write(zap.Uint64("seqence", received))
		}

		err = s.handleFunc(rs, msg, received)
		if err != nil {
			logger.Error("session handle failed, close this session",
				zap.Error(err))
			return err
		}
	}
}

func (s *server) addSession(session IOSession) bool {
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

func (s *server) deleteSession(session IOSession) bool {
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

func (s *server) nextID() uint64 {
	return atomic.AddUint64(&s.atomic.id, 1)
}

func (s *server) isStarted() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.mu.running
}

func parseAdddress(address string) (string, string, error) {
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
