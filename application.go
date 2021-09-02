package goetty

import (
	"errors"
	"fmt"
	"io"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

var (
	stateReadyToStart = int32(0)
	stateStarting     = int32(1)
	stateStarted      = int32(2)
	stateStopping     = int32(3)
	stateStopped      = int32(4)
)

// NetApplication is a network based application
type NetApplication interface {
	// Start start the transport server
	Start() error
	// Stop stop the transport server
	Stop() error
	// GetSession get session
	GetSession(uint64) (IOSession, error)
	// Broadcast broadcast msg to all sessions
	Broadcast(msg interface{}) error
}

type sessionMap struct {
	sync.RWMutex
	sessions map[uint64]IOSession
}

type server struct {
	id         uint64
	opts       *appOptions
	listener   net.Listener
	startCh    chan struct{}
	state      int32
	sessions   map[uint64]*sessionMap
	handleFunc func(IOSession, interface{}, uint64) error
}

// NewApplication returns a net application with listener
func NewApplication(listener net.Listener, handleFunc func(IOSession, interface{}, uint64) error, opts ...AppOption) (NetApplication, error) {
	s := &server{
		listener:   listener,
		handleFunc: handleFunc,
		opts: &appOptions{
			sessionOpts: &options{},
		},
		startCh: make(chan struct{}, 1),
	}

	for _, opt := range opts {
		opt(s.opts)
	}

	s.opts.adjust()
	s.sessions = make(map[uint64]*sessionMap, s.opts.sessionBucketSize)
	for i := uint64(0); i < s.opts.sessionBucketSize; i++ {
		s.sessions[i] = &sessionMap{
			sessions: make(map[uint64]IOSession),
		}
	}
	return s, nil
}

// NewTCPApplication returns a net application
func NewTCPApplication(addr string, handleFunc func(IOSession, interface{}, uint64) error, opts ...AppOption) (NetApplication, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	return NewApplication(listener, handleFunc, opts...)
}

func (s *server) Start() error {
	old := s.getState()
	switch old {
	case stateStarting:
		return errors.New("server is in starting")
	case stateStopping:
		return errors.New("server is in stopping")
	case stateStopped:
		return errors.New("server is stopped")
	case stateStarted:
		return nil
	case stateReadyToStart:
		break
	}

	if !atomic.CompareAndSwapInt32(&s.state, stateReadyToStart, stateStarting) {
		current := s.getState()
		if current == stateStarted {
			return nil
		}

		return fmt.Errorf("error state %d", current)
	}

	c := make(chan error)
	go func() {
		c <- s.doStart()
	}()

	select {
	case <-s.startCh:
		atomic.StoreInt32(&s.state, stateStarted)
		s.opts.sessionOpts.logger.Info("goetty application started")
		return nil
	case err := <-c:
		return err
	}
}

func (s *server) Stop() error {
	old := s.getState()
	switch old {
	case stateStarting:
		return errors.New("server not started")
	case stateStopping:
		return errors.New("server is in stopping")
	case stateStopped:
		return nil
	case stateReadyToStart:
		return errors.New("server is not start")
	case stateStarted:
		break
	}

	if !atomic.CompareAndSwapInt32(&s.state, stateStarted, stateStopping) {
		current := s.getState()
		if current == stateStopped {
			return nil
		}

		return fmt.Errorf("error state %d", current)
	}

	s.listener.Close()
	close(s.startCh)
	atomic.StoreInt32(&s.state, stateStopped)
	return nil
}

func (s *server) GetSession(id uint64) (IOSession, error) {
	state := s.getState()
	if state != stateStarted {
		return nil, errors.New("server is not started")
	}

	m := s.sessions[id%s.opts.sessionBucketSize]
	m.RLock()
	session := m.sessions[id]
	m.RUnlock()
	return session, nil
}

func (s *server) Broadcast(msg interface{}) error {
	state := s.getState()
	if state != stateStarted {
		return errors.New("server is not started")
	}

	for _, m := range s.sessions {
		m.RLock()
		for _, rs := range m.sessions {
			rs.WriteAndFlush(msg)
		}
		m.RUnlock()
	}

	return nil
}

func (s *server) doStart() error {
	s.startCh <- struct{}{}
	var tempDelay time.Duration
	for {
		conn, err := s.listener.Accept()
		state := s.getState()
		switch state {
		case stateStopping:
		case stateStopped:
			if nil != conn {
				conn.Close()
			}
			return nil
		}

		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
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
			return err
		}
		tempDelay = 0

		rs := newBaseIOWithOptions(s.nextID(), conn, s.opts.sessionOpts)
		s.addSession(rs)

		go func() {
			defer func() {
				if err := recover(); err != nil {
					const size = 64 << 10
					rBuf := make([]byte, size)
					rBuf = rBuf[:runtime.Stack(rBuf, false)]
					s.opts.sessionOpts.logger.Error("connection painc",
						zap.Any("err", err),
						zap.String("stack", string(rBuf)))
				}
			}()

			defer func() {
				s.deleteSession(rs)
				rs.Close()
				if s.opts.aware != nil {
					s.opts.aware.Closed(rs)
				}
			}()
			if s.opts.aware != nil {
				s.opts.aware.Created(rs)
			}
			s.doConnection(rs)
		}()
	}
}

func (s *server) doConnection(rs IOSession) error {
	logger := s.opts.sessionOpts.logger.With(zap.Uint64("session-id", rs.ID()),
		zap.String("addr", rs.RemoteAddr()))

	logger.Info("session connected")

	received := uint64(0)
	for {
		msg, err := rs.Read()
		if err != nil {
			if err == io.EOF {
				return nil
			}

			logger.Info("session read failed",
				zap.Error(err))
			return err
		}

		logger.Debug("session read", zap.Any("msg", msg))
		received++

		err = s.handleFunc(rs, msg, received)
		if err != nil {
			if s.opts.errorMsgFactory == nil {
				logger.Error("session handle failed, close this session",
					zap.Error(err))
				return err
			}

			rs.Write(s.opts.errorMsgFactory(rs, msg, err))
		}
	}
}

func (s *server) addSession(session IOSession) {
	m := s.sessions[session.ID()%s.opts.sessionBucketSize]
	m.Lock()
	m.sessions[session.ID()] = session
	m.Unlock()
}

func (s *server) deleteSession(session IOSession) {
	m := s.sessions[session.ID()%s.opts.sessionBucketSize]
	m.Lock()
	delete(m.sessions, session.ID())
	m.Unlock()
}

func (s *server) nextID() uint64 {
	return atomic.AddUint64(&s.id, 1)
}

func (s *server) getState() int32 {
	return atomic.LoadInt32(&s.state)
}
