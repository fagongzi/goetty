package transport

import (
	"io"
	"sync"

	"github.com/fagongzi/goetty"
)

// Server is the transport server
type Server interface {
	// Start start the transport server
	Start() error
	// Stop stop the transport server
	Stop() error
}

type server struct {
	addr string

	opts       *options
	svr        *goetty.Server
	handleFunc func(*Session, interface{}) error
	sessions   sync.Map // interface{} -> *util.Session
}

// New create a server
func New(addr string, handleFunc func(*Session, interface{}) error, opts ...Option) Server {
	s := &server{
		addr:       addr,
		handleFunc: handleFunc,
		opts:       &options{},
	}

	for _, opt := range opts {
		opt(s.opts)
	}

	s.opts.adjust()

	s.svr = goetty.NewServer(addr,
		goetty.WithServerDecoder(s.opts.decoder),
		goetty.WithServerEncoder(s.opts.encoder))

	return s
}

func (s *server) Start() error {
	s.opts.logger.Infof("api server start at %s", s.addr)
	c := make(chan error)
	go func() {
		c <- s.svr.Start(s.doConnection)
	}()

	select {
	case <-s.svr.Started():
		return nil
	case err := <-c:
		return err
	}
}

func (s *server) Stop() error {
	s.svr.Stop()
	return nil
}

func (s *server) doConnection(conn goetty.IOSession) error {
	rs := NewSession(conn, s.opts.factory(32), s.opts.logger, s.opts.respReleaseFunc)
	s.sessions.Store(rs.ID, rs)
	s.opts.logger.Infof("session %d[%s] connected",
		rs.ID,
		rs.Addr)

	defer func() {
		s.sessions.Delete(rs.ID)
		rs.Close()
	}()

	for {
		req, err := conn.Read()
		if err != nil {
			if err == io.EOF {
				return nil
			}

			s.opts.logger.Errorf("session %d[%s] read failed with %+v",
				rs.ID,
				rs.Addr,
				err)
			return err
		}

		s.opts.logger.Debugf("session %d[%s] read %+v",
			rs.ID,
			rs.Addr,
			req)

		err = s.handleFunc(rs, req)
		if err != nil {
			if s.opts.errorResponseFactory == nil {
				s.opts.logger.Errorf("session %d[%s] handle failed with %+v, close session",
					rs.ID,
					rs.Addr,
					err)
				return err
			}

			rs.OnResp(s.opts.errorResponseFactory(req, err))
		}
	}
}
