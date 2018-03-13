package goetty

import (
	"errors"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	// ErrWrite write error
	ErrWrite = errors.New("goetty.net: Write failed")
	// ErrEmptyServers empty server error
	ErrEmptyServers = errors.New("goetty.Connector: Empty servers pool")
	// ErrIllegalState illegal state error
	ErrIllegalState = errors.New("goetty.Connector: Not connected")
)

type connector struct {
	sync.RWMutex

	opts        *clientOptions
	addr        string
	conn        net.Conn
	closed      int32
	lastTimeout Timeout
	attrs       map[string]interface{}
	in          *ByteBuf
	out         *ByteBuf
}

// NewConnector create a new connector with opts
func NewConnector(svrAddr string, opts ...ClientOption) IOSession {
	sopts := &clientOptions{}
	for _, opt := range opts {
		opt(sopts)
	}
	sopts.adjust()

	return &connector{
		addr:  svrAddr,
		in:    NewByteBuf(sopts.readBufSize),
		out:   NewByteBuf(sopts.writeBufSize),
		opts:  sopts,
		attrs: make(map[string]interface{}),
	}
}

// InBuf returns internal bytebuf that used for read from client
func (c *connector) InBuf() *ByteBuf {
	return c.in
}

// OutBuf returns internal bytebuf that used for write to client
func (c *connector) OutBuf() *ByteBuf {
	return c.out
}

// SetAttr add a attr on session
func (c *connector) SetAttr(key string, value interface{}) {
	c.Lock()
	c.attrs[key] = value
	c.Unlock()
}

// GetAttr get attr from session
func (c *connector) GetAttr(key string) interface{} {
	c.RLock()
	v := c.attrs[key]
	c.RUnlock()
	return v
}

// ID get id
func (c *connector) ID() interface{} {
	return 0
}

// Hash get hash value use id
func (c *connector) Hash() int {
	return 0
}

func (c *connector) Connect() (bool, error) {
	e := c.Close() // Close current connection

	if e != nil {
		return false, e
	}

	conn, e := net.DialTimeout("tcp", c.addr, c.opts.connectTimeout)

	if nil != e {
		return false, e
	}

	conn.(*net.TCPConn).SetNoDelay(true)
	conn.(*net.TCPConn).SetLinger(0)
	c.conn = conn
	atomic.StoreInt32(&c.closed, 0)
	c.bindWriteTimeout()

	return true, nil
}

// Close close
func (c *connector) Close() error {
	if nil != c.conn {
		err := c.conn.Close()
		if err != nil {
			c.reset()
			return err
		}
	}

	c.reset()
	return nil
}

// IsConnected is connected
func (c *connector) IsConnected() bool {
	return nil != c.conn && atomic.LoadInt32(&c.closed) == 0
}

func (c *connector) reset() {
	atomic.StoreInt32(&c.closed, 1)
	c.conn = nil
	c.in.Clear()
	c.out.Clear()
}

// Read read data from server, block until a msg arrived or  get a error
func (c *connector) Read() (interface{}, error) {
	return c.ReadTimeout(0)
}

// ReadTimeout read data from server with a timeout duration
func (c *connector) ReadTimeout(timeout time.Duration) (interface{}, error) {
	if !c.IsConnected() {
		return nil, ErrIllegalState
	}

	var msg interface{}
	var err error
	var complete bool

	for {
		if c.in.Readable() > 0 {
			complete, msg, err = c.opts.decoder.Decode(c.in)

			if !complete && err == nil {
				complete, msg, err = c.readFromConn(timeout)
			}
		} else {
			complete, msg, err = c.readFromConn(timeout)
		}

		if nil != err {
			c.in.Clear()
			return nil, err
		}

		if complete {
			break
		}
	}

	if c.in.Readable() == 0 {
		c.in.Clear()
	}

	return msg, err
}

// Write write a msg to server
func (c *connector) Write(msg interface{}) error {
	return c.write(msg, false)
}

// WriteAndFlush write a msg to server
func (c *connector) WriteAndFlush(msg interface{}) error {
	return c.write(msg, true)
}

func (c *connector) write(msg interface{}, flush bool) error {
	err := c.opts.encoder.Encode(msg, c.out)

	if err != nil {
		return err
	}

	if flush {
		return c.Flush()
	}

	return nil
}

// Flush writes bytes that in the internal bytebuf
func (c *connector) Flush() error {
	buf := c.out

	written := 0
	all := buf.Readable()
	for {
		if written == all {
			break
		}

		n, err := c.conn.Write(buf.buf[buf.readerIndex+written : buf.writerIndex])
		if err != nil {
			c.writeRelease()
			return err
		}

		written += n
	}

	c.writeRelease()
	return nil
}

// RemoteAddr get remote address
func (c *connector) RemoteAddr() string {
	if nil != c.conn {
		return c.conn.RemoteAddr().String()
	}

	return ""
}

// RemoteIP return remote ip address
func (c *connector) RemoteIP() string {
	addr := c.RemoteAddr()
	if addr == "" {
		return ""
	}

	return strings.Split(addr, ":")[0]
}

func (c *connector) readFromConn(timeout time.Duration) (bool, interface{}, error) {
	if 0 != timeout {
		c.conn.SetReadDeadline(time.Now().Add(timeout))
	}

	_, err := c.in.ReadFrom(c.conn)

	if err != nil {
		return false, nil, err
	}

	return c.opts.decoder.Decode(c.in)
}

func (c *connector) writeRelease() {
	c.out.Clear()
	c.bindWriteTimeout()
}

func (c *connector) bindWriteTimeout() {
	if c.opts.writeTimeoutHandler != nil {
		c.lastTimeout.Stop()
		c.lastTimeout, _ = c.opts.timeWheel.Schedule(c.opts.writeTimeout, c.writeTimeout, nil)
	}
}

func (c *connector) cancelWriteTimeout() {
	if c.opts.writeTimeoutHandler != nil {
		c.lastTimeout.Stop()
	}
}

func (c *connector) writeTimeout(arg interface{}) {
	if c.opts.writeTimeoutHandler != nil {
		c.opts.writeTimeoutHandler(c.addr, c)
	}
}
