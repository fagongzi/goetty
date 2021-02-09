package transport

import (
	"errors"
	"sync"

	"github.com/fagongzi/goetty"
)

var (
	stopFlag  = &struct{}{}
	errClosed = errors.New("session closed")
)

// Session session
type Session struct {
	ID   interface{}
	Addr string

	log         Logger
	resps       Queue
	conn        goetty.IOSession
	releaseFunc func(interface{})
	stopOnce    sync.Once
	stopped     chan struct{}
}

// NewSession create a client session
func NewSession(conn goetty.IOSession, resps Queue, log Logger, releaseFunc func(interface{})) *Session {
	s := &Session{
		ID:          conn.ID(),
		Addr:        conn.RemoteAddr(),
		resps:       resps,
		conn:        conn,
		releaseFunc: releaseFunc,
		stopped:     make(chan struct{}),
		log:         log,
	}

	go s.writeLoop()
	return s
}

// Close close the client session
func (s *Session) Close() {
	s.resps.Put(stopFlag)
	<-s.stopped
	s.log.Infof("session %d[%s] closed",
		s.ID,
		s.Addr)
}

// Closed returns true if the session is closed
func (s *Session) Closed() bool {
	return s.resps.Disposed()
}

// OnResp receive a response
func (s *Session) OnResp(resp interface{}) error {
	if s != nil {
		return s.resps.Put(resp)
	}

	return errClosed
}

func (s *Session) doClose() {
	s.stopOnce.Do(func() {
		s.resps.Dispose()
		s.conn.Close()
		s.stopped <- struct{}{}
	})
}

func (s *Session) releaseResp(resp interface{}) {
	if s.releaseFunc != nil && resp != nil {
		s.releaseFunc(resp)
	}
}

func (s *Session) writeLoop() {
	items := make([]interface{}, 16, 16)
	for {
		n, err := s.resps.Get(16, items)
		if nil != err {
			s.log.Fatalf("BUG: can not failed")
		}

		for i := int64(0); i < n; i++ {
			if items[i] == stopFlag {
				s.doClose()
				return
			}

			s.conn.Write(items[i])
			s.releaseResp(items[i])
		}

		s.conn.Flush()
	}
}
