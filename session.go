package goetty

import (
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fagongzi/goetty/v2/buf"
	"github.com/fagongzi/goetty/v2/codec"
	"go.uber.org/zap"
)

var (
	// ErrIllegalState illegal state error
	ErrIllegalState = errors.New("illegal state")
	// ErrDisableConnect disable to connect
	ErrDisableConnect = errors.New("io session is disable to connect")

	stateReadyToConnect int32 = 0
	stateConnectting    int32 = 1
	stateConnected      int32 = 2
	stateClosing        int32 = 3
)

// WriteOptions write options
type WriteOptions struct {
	// Timeout deadline for write
	Timeout time.Duration
	// Flush flush data to net.Conn
	Flush bool
}

// ReadOptions read options
type ReadOptions struct {
	// Timeout deadline for read
	Timeout time.Duration
}

// Option option to create IOSession
type Option func(*baseIO)

// WithDisableReleaseOutBuf set disable release buf
func WithDisableReleaseOutBuf() Option {
	return func(bio *baseIO) {
		bio.options.disableReleaseOut = true
	}
}

// WithSessionLogger set logger for IOSession
func WithSessionLogger(logger *zap.Logger) Option {
	return func(bio *baseIO) {
		bio.logger = logger
	}
}

// WithSessionAllocator set mem allocator to build in and out ByteBuf
func WithSessionAllocator(allocator buf.Allocator) Option {
	return func(bio *baseIO) {
		bio.options.allocator = allocator
	}
}

// WithSessionConnOptionFunc set conn options func
func WithSessionConnOptionFunc(connOptionFunc func(net.Conn)) Option {
	return func(bio *baseIO) {
		bio.options.connOptionFunc = connOptionFunc
	}
}

// WithSessionCodec set codec for IOSession
func WithSessionCodec(codec codec.Codec) Option {
	return func(bio *baseIO) {
		bio.options.codec = codec
	}
}

// WithSessionRWBUfferSize set read/write buf size for IOSession
func WithSessionRWBUfferSize(read, write int) Option {
	return func(bio *baseIO) {
		bio.options.readBufSize = read
		bio.options.writeBufSize = write
	}
}

// WithSessionConn set IOSession's net.Conn
func WithSessionConn(id uint64, conn net.Conn) Option {
	return func(bio *baseIO) {
		bio.conn = conn
		bio.id = id
	}
}

// WithSessionReleaseMsgFunc set a func to release message once the message encode into the write buf
func WithSessionReleaseMsgFunc(value func(any)) Option {
	return func(bio *baseIO) {
		bio.options.releaseMsgFunc = value
	}
}

// IOSession session
type IOSession interface {
	// ID sessino id
	ID() uint64
	// Connect connect to address, only used at client-side
	Connect(addr string, timeout time.Duration) (bool, error)
	// Close close the session
	Close() error
	// Connected returns true if connection is ok
	Connected() bool
	// Read read packet from connection
	Read(option ReadOptions) (interface{}, error)
	// Write encodes the msg into a []byte into the buffer according to the codec.Encode.
	// If flush is set to flase, the data will not be written to the underlying socket.
	Write(msg interface{}, options WriteOptions) error
	// Flush flush the out buffer
	Flush(timeout time.Duration) error
	// InBuf connection read buffer
	InBuf() *buf.ByteBuf
	// OutBuf connection out buffer
	OutBuf() *buf.ByteBuf
	// SetAttr set attr
	SetAttr(key string, value interface{})
	// GetAttr read attr
	GetAttr(key string) interface{}
	// RemoteAddr returns remote address, include ip and port
	RemoteAddr() string
	// RemoteIP returns remote address, only ip
	RemoteIP() string
	// RawConn returns the raw connection
	RawConn() (net.Conn, error)
}

type baseIO struct {
	id                    uint64
	state                 int32
	conn                  net.Conn
	localAddr, remoteAddr string
	in                    *buf.ByteBuf
	out                   *buf.ByteBuf
	attrs                 sync.Map
	disableConnect        bool
	logger                *zap.Logger
	readCopyBuf           []byte
	writeCopyBuf          []byte

	options struct {
		codec                             codec.Codec
		readBufSize, writeBufSize         int
		readCopyBufSize, writeCopyBufSize int
		connOptionFunc                    func(net.Conn)
		releaseMsgFunc                    func(interface{})
		disableReleaseOut                 bool
		allocator                         buf.Allocator
	}
}

// NewIOSession create a new io session
func NewIOSession(opts ...Option) IOSession {
	bio := &baseIO{}
	for _, opt := range opts {
		opt(bio)
	}
	bio.adjust()

	bio.readCopyBuf = make([]byte, bio.options.readCopyBufSize)
	bio.writeCopyBuf = make([]byte, bio.options.writeCopyBufSize)
	if bio.conn != nil {
		bio.initConn()
		bio.disableConnect = true
	}
	return bio
}

func (bio *baseIO) adjust() {
	bio.logger = adjustLogger(bio.logger)
	if bio.options.readBufSize == 0 {
		bio.options.readBufSize = DefaultReadBuf
	}
	if bio.options.readCopyBufSize == 0 {
		bio.options.readCopyBufSize = DefaultReadCopyBuf
	}
	if bio.options.writeBufSize == 0 {
		bio.options.writeBufSize = DefaultWriteBuf
	}
	if bio.options.writeCopyBufSize == 0 {
		bio.options.writeCopyBufSize = DefaultWriteCopyBuf
	}
	if bio.options.releaseMsgFunc == nil {
		bio.options.releaseMsgFunc = func(interface{}) {}
	}
	if bio.options.connOptionFunc == nil {
		bio.options.connOptionFunc = func(net.Conn) {}
	}
}

func (bio *baseIO) ID() uint64 {
	return bio.id
}

func (bio *baseIO) Connect(addressWithNetwork string, timeout time.Duration) (bool, error) {
	network, address, err := parseAdddress(addressWithNetwork)
	if err != nil {
		return false, err
	}

	if bio.disableConnect {
		return false, ErrDisableConnect
	}

	old := bio.getState()
	switch old {
	case stateReadyToConnect:
		break
	case stateClosing:
		return false, fmt.Errorf("the session is closing in other goroutine")
	case stateConnectting:
		return false, fmt.Errorf("the session is connecting in other goroutine")
	case stateConnected:
		return true, nil
	}

	// only stateReadyToConnect state can connect
	if !atomic.CompareAndSwapInt32(&bio.state, stateReadyToConnect, stateConnectting) {
		current := bio.getState()
		if current == stateConnected {
			return true, nil
		}

		return false, fmt.Errorf("the session is closing or connecting is other goroutine")
	}

	conn, err := net.DialTimeout(network, address, timeout)
	if nil != err {
		atomic.StoreInt32(&bio.state, stateReadyToConnect)
		return false, err
	}

	bio.conn = conn
	bio.initConn()
	return true, nil
}

func (bio *baseIO) Connected() bool {
	return bio.getState() == stateConnected
}

func (bio *baseIO) Close() error {
	old := bio.getState()
	switch old {
	case stateReadyToConnect:
		return nil
	case stateClosing:
		return fmt.Errorf("the session is closing in other goroutine")
	case stateConnectting:
		return fmt.Errorf("the session is connecting in other goroutine")
	case stateConnected:
		break
	}

	// only stateConnected state close
	if !atomic.CompareAndSwapInt32(&bio.state, stateConnected, stateClosing) {
		current := bio.getState()
		if current == stateReadyToConnect {
			return nil
		}

		return fmt.Errorf("the session is closing or connecting is other goroutine")
	}

	bio.closeConn()
	if !bio.options.disableReleaseOut {
		bio.out.Release()
	}
	atomic.StoreInt32(&bio.state, stateReadyToConnect)
	return nil
}

func (bio *baseIO) Read(options ReadOptions) (interface{}, error) {
	for {
		if !bio.Connected() {
			return nil, ErrIllegalState
		}

		var msg interface{}
		var err error
		var complete bool
		for {
			if bio.in.Readable() > 0 {
				msg, complete, err = bio.options.codec.Decode(bio.in)
				if !complete && err == nil {
					msg, complete, err = bio.readFromConn(options.Timeout)
				}
			} else {
				bio.in.Reset()
				msg, complete, err = bio.readFromConn(options.Timeout)
			}

			if nil != err {
				bio.in.Release()
				return nil, err
			}

			if complete {
				if bio.in.Readable() == 0 {
					bio.in.Reset()
				}

				return msg, nil
			}
		}
	}
}

func (bio *baseIO) Write(msg interface{}, options WriteOptions) error {
	if !bio.Connected() {
		return ErrIllegalState
	}

	err := bio.options.codec.Encode(msg, bio.out, bio.conn)
	bio.options.releaseMsgFunc(msg)
	if err != nil {
		return err
	}

	if options.Flush && bio.out.Readable() > 0 {
		err = bio.Flush(options.Timeout)
		if err != nil {
			return err
		}
	}

	return nil
}

func (bio *baseIO) Flush(timeout time.Duration) error {
	defer bio.out.Reset()
	if !bio.Connected() {
		return ErrIllegalState
	}

	if timeout != 0 {
		bio.conn.SetWriteDeadline(time.Now().Add(timeout))
	} else {
		bio.conn.SetWriteDeadline(time.Time{})
	}

	_, err := io.CopyBuffer(bio.conn, bio.out, bio.writeCopyBuf)
	if err == nil || err == io.EOF {
		return nil
	}
	return err
}

func (bio *baseIO) RemoteAddr() string {
	return bio.remoteAddr
}

func (bio *baseIO) RemoteIP() string {
	if bio.remoteAddr == "" {
		return ""
	}
	return strings.Split(bio.remoteAddr, ":")[0]
}

func (bio *baseIO) InBuf() *buf.ByteBuf {
	return bio.in
}

func (bio *baseIO) OutBuf() *buf.ByteBuf {
	return bio.out
}

func (bio *baseIO) SetAttr(key string, value interface{}) {
	bio.attrs.Store(key, value)
}

func (bio *baseIO) GetAttr(key string) interface{} {
	if v, ok := bio.attrs.Load(key); ok {
		return v
	}

	return nil
}

func (bio *baseIO) RawConn() (net.Conn, error) {
	if !bio.Connected() {
		return nil, ErrIllegalState
	}
	return bio.conn, nil
}

func (bio *baseIO) readFromConn(timeout time.Duration) (any, bool, error) {
	if timeout != 0 {
		bio.conn.SetReadDeadline(time.Now().Add(timeout))
	} else {
		bio.conn.SetReadDeadline(time.Time{})
	}

	n, err := io.CopyBuffer(bio.in, bio.conn, bio.readCopyBuf)
	if err != nil {
		return nil, false, err
	}
	if n == 0 {
		return nil, false, io.EOF
	}
	return bio.options.codec.Decode(bio.in)
}

func (bio *baseIO) closeConn() {
	if bio.conn != nil {
		bio.conn.Close()
	}
}

func (bio *baseIO) getState() int32 {
	return atomic.LoadInt32(&bio.state)
}

func (bio *baseIO) initConn() {
	bio.remoteAddr = bio.conn.RemoteAddr().String()
	bio.localAddr = bio.conn.LocalAddr().String()
	bio.in = buf.NewByteBuf(bio.options.readBufSize, buf.WithMemAllocator(bio.options.allocator))
	bio.out = buf.NewByteBuf(bio.options.writeBufSize, buf.WithMemAllocator(bio.options.allocator))
	bio.logger = adjustLogger(bio.logger).Named("io-session").With(zap.Uint64("id", bio.id),
		zap.String("local-address", bio.localAddr),
		zap.String("remote-address", bio.remoteAddr))
	atomic.StoreInt32(&bio.state, stateConnected)
}
