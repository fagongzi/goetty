package example

import (
	"log"

	"github.com/fagongzi/goetty/v2"
	"github.com/fagongzi/goetty/v2/codec/simple"
)

// EchoServer echo server
type EchoServer struct {
	addr string
	app  goetty.NetApplication
}

// NewEchoServer create new server
func NewEchoServer(addr string) *EchoServer {
	svr := &EchoServer{}
	app, err := goetty.NewApplication(addr, svr.handle,
		goetty.WithAppSessionOptions(goetty.WithSessionCodec(simple.NewStringCodec())))
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
	return s.app.Start()
}

// Stop stop
func (s *EchoServer) Stop() error {
	return s.app.Stop()
}

func (s *EchoServer) handle(session goetty.IOSession, message any, received uint64) error {
	log.Printf("received %+v from %s, already received %d messages",
		message,
		session.RemoteAddress(),
		received)
	return session.Write(message, goetty.WriteOptions{Flush: true})
}
