package example

import (
	"github.com/fagongzi/goetty"
)

type EchoClient struct {
	serverAddr string
	conn       *goetty.Connector
}

func NewEchoClient(serverAddr string) (*EchoClient, error) {
	cnf := &Conf{
		Addr: serverAddr,
		TimeoutConnectToServer: time.Second * 3,
	}

	c := &EchoClient{
		serverAddr: serverAddr,
		conn:       NewConnector(cnf, NewIntLengthFieldBasedDecoder(&StringDecoder{}), &StringEncoder{}),
	}

	// if you want to send heartbeat to server, you can set conf as below, otherwise not set

	// create a timewheel to calc timeout
	tw := NewHashedTimeWheel(time.Second, 60, 3)
	tw.Start()

	cnf.TimeoutWrite = time.Second * 3
	cnf.TimeWheel = tw
	cnf.WriteTimeoutFn = c.writeHeartbeat

	_, err := c.conn.Connect()

	return c, err
}

func (self *EchoClient) writeHeartbeat(serverAddr string, conn *goetty.Connector) {
	self.SendMsg("this is a heartbeat msg")
}

func (self *EchoClient) SendMsg(msg string) error {
	return self.conn.Write(msg)
}

func (self *EchoClient) ReadLoop() error {
	// start loop to read msg from server
	for {
		msg, err := connector.Read() // if you want set a read deadline, you can use 'connector.ReadTimeout(timeout)'
		if err != nil {
			fmt.Printf("read msg from server<%s> failure", self.serverAddr)
			return err
		}

		fmt.Printf("receive a msg<%s> from <%s>", msg, self.serverAddr)
	}

	return nil
}
