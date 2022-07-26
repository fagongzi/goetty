package example

import (
	"log"
	"time"

	"github.com/fagongzi/goetty/v2"
	"github.com/fagongzi/goetty/v2/codec/simple"
)

// EchoClient echo client
type EchoClient struct {
	serverAddr string
	conn       goetty.IOSession
}

// NewEchoClient new client
func NewEchoClient(serverAddr string) (*EchoClient, error) {
	c := &EchoClient{
		serverAddr: serverAddr,
	}

	c.conn = goetty.NewIOSession(goetty.WithSessionCodec(simple.NewStringCodec()))
	err := c.conn.Connect(serverAddr, time.Second*3)
	return c, err
}

// SendMsg send msg to server
func (c *EchoClient) SendMsg(msg string) error {
	return c.conn.Write(msg, goetty.WriteOptions{Flush: true})
}

// ReadLoop read loop
func (c *EchoClient) ReadLoop() error {
	// start loop to read msg from server
	for {
		msg, err := c.conn.Read(goetty.ReadOptions{})
		if err != nil {
			log.Printf("read failed with %+v", err)
			return err
		}

		log.Printf("received %+v", msg)
	}
}
