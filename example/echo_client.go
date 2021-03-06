package example

import (
	"log"
	"time"

	"github.com/fagongzi/goetty"
	"github.com/fagongzi/goetty/codec/simple"
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

	encoder, decoder := simple.NewStringCodec()
	c.conn = goetty.NewIOSession(goetty.WithCodec(encoder, decoder))
	_, err := c.conn.Connect(serverAddr, time.Second*3)
	return c, err
}

// SendMsg send msg to server
func (c *EchoClient) SendMsg(msg string) error {
	return c.conn.WriteAndFlush(msg)
}

// ReadLoop read loop
func (c *EchoClient) ReadLoop() error {
	// start loop to read msg from server
	for {
		msg, err := c.conn.Read() // if you want set a read deadline, you can use 'WithTimeout option'
		if err != nil {
			log.Printf("read failed with %+v", err)
			return err
		}

		log.Printf("received %+v", msg)
	}
}
