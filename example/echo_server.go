package example

import (
	"log"

	"github.com/fagongzi/goetty"
	"github.com/fagongzi/goetty/codec/simple"
)

// EchoServer echo server
type EchoServer struct {
	addr string
	app  goetty.NetApplication
}

// NewEchoServer create new server
func NewEchoServer(addr string) *EchoServer {
	svr := &EchoServer{}
	encoder, decoder := simple.NewStringCodec()
	app, err := goetty.NewApplication(addr, svr.handle,
		goetty.WithAppSessionOptions(goetty.WithCodec(encoder, decoder)))
	if err != nil {
		log.Panicf("start server failed with %+v", err)
	}

	return &EchoServer{
		addr: addr,
		app:  app,
	}
}

// Start start
func (s *EchoServer) Start() error {
	return s.Start()
}

// Stop stop
func (s *EchoServer) Stop() error {
	return s.Stop()
}

func (s *EchoServer) handle(session goetty.IOSession, msg interface{}, received uint64) error {
	log.Printf("received %+v from %s, already received %d msgs",
		msg,
		session.RemoteAddr(),
		received)
	return session.WriteAndFlush(msg)
}
