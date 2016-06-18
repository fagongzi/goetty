package example

import (
	"fmt"
	"github.com/fagongzi/goetty"
)

type EchoServer struct {
	addr   string
	server *goetty.Server
}

func NewEchoServer(addr string) *EchoServer {
	return &EchoServer{
		addr:   addr,
		server: goetty.NewServer(addr, NewIntLengthFieldBasedDecoder(&StringDecoder{}), &StringEncoder{}, NewInt64IdGenerator()),
	}
}

func (self *EchoServer) Serve() error {
	return self.server.Serve(loopFn)
}

func (self *EchoServer) doConnection(session goetty.IOSession) {
	defer session.Close() // close the connection

	fmt.Printf("A new connection from <%s>", session.RemoteAddr())

	// start loop for read msg from this connection
	for {
		msg, err := session.Read() // if you want set a read deadline, you can use 'session.ReadTimeout(timeout)'
		if err != nil {
			return err
		}

		fmt.Printf("receive a msg<%s> from <%s>", msg, session.RemoteAddr())

		// echo msg back
		session.Write(msg)
	}
}
