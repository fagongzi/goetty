package goetty

import (
	"errors"
	"net"
	"sync"
	"time"
)

var (
	WriteErr        = errors.New("goetty.net: Write failed")
	EmptyServersErr = errors.New("goetty.Connector: Empty servers pool")
	IllegalStateErr = errors.New("goetty.Connector: Not connected")
)

type Conf struct {
	Addr                   string
	TimeoutConnectToServer time.Duration
	TimeoutWrite           time.Duration
	TimeoutRead            time.Duration
	TimeWheel              *HashedTimeWheel

	ReadTimeoutFn  func(string, *Connector)
	WriteTimeoutFn func(string, *Connector)
}

type Connector struct {
	cnf *Conf

	decoder Decoder
	encoder Encoder

	conn         net.Conn
	connected    bool
	writeBufSize int

	timeoutReadKey, timeoutWriteKey string

	in  *ByteBuf
	out sync.Pool
}

func NewConnector(cnf *Conf, decoder Decoder, encoder Encoder) *Connector {
	return NewConnectorSize(cnf, decoder, encoder, BUF_READ_SIZE, BUF_WRITE_SIZE)
}

func NewConnectorSize(cnf *Conf, decoder Decoder, encoder Encoder, readBufSize, writeBufSize int) *Connector {
	return &Connector{
		cnf:          cnf,
		in:           NewByteBuf(readBufSize),
		writeBufSize: writeBufSize,
		decoder:      decoder,
		encoder:      encoder,
	}
}

func (c *Connector) Connect() (bool, error) {
	e := c.Close() // Close current connection

	if e != nil {
		return false, e
	}

	conn, e := net.DialTimeout("tcp", c.cnf.Addr, c.cnf.TimeoutConnectToServer)

	if nil != e {
		return false, e
	}

	c.conn = conn
	c.connected = true

	c.bindWriteTimeout()

	return true, nil
}

func (c *Connector) Close() error {
	defer c.reset()

	if nil != c.conn {
		return c.conn.Close()
	}

	return nil
}

func (c *Connector) IsConnected() bool {
	return nil != c.conn && c.connected
}

func (c *Connector) reset() {
	c.connected = false
	c.conn = nil
}

func (c *Connector) Read() (interface{}, error) {
	if !c.IsConnected() {
		return nil, IllegalStateErr
	}

	defer c.in.Clear()

	for {
		_, err := c.in.ReadFrom(c.conn)

		if err != nil {
			return nil, err
		}

		complete, msg, err := c.decoder.Decode(c.in)

		if nil != err {
			return nil, err
		}

		if complete {
			return msg, nil
		}
	}

	return nil, nil
}

func (c *Connector) Write(msg interface{}) error {
	if c.IsConnected() {
		defer c.bindWriteTimeout()

		buf, ok := c.out.Get().(*ByteBuf)

		if !ok {
			buf = NewByteBuf(c.writeBufSize)
		}

		defer func() {
			buf.Clear()
			if !ok {
				c.out.Put(buf)
			}
		}()

		err := c.encoder.Encode(msg, buf)

		if err != nil {
			return err
		}

		_, bytes, _ := buf.ReadAll()

		n, err := c.conn.Write(bytes)

		if err != nil {
			return err
		}

		c.cancelWriteTimeout()

		if n != len(bytes) {
			return WriteErr
		}

		return nil
	}

	return IllegalStateErr
}

func (c *Connector) bindWriteTimeout() {
	c.timeoutWriteKey = c.cnf.TimeWheel.Add(c.cnf.TimeoutWrite, c.writeTimeout)
}

func (c *Connector) cancelWriteTimeout() {
	c.cnf.TimeWheel.Cancel(c.timeoutWriteKey)
}

func (c *Connector) writeTimeout(key string) {
	if c.timeoutWriteKey == key && c.cnf.WriteTimeoutFn != nil {
		c.cnf.WriteTimeoutFn(c.cnf.Addr, c)
	}
}
