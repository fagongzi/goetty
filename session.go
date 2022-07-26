package goetty

import (
	"errors"
	"fmt"
	"io"
	"net"
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
	stateConnecting     int32 = 1
	stateConnected      int32 = 2
	stateClosed         int32 = 3
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

// WithSessionAware set IOSession's session aware
func WithSessionAware(value IOSessionAware) Option {
	return func(bio *baseIO) {
		bio.options.aware = value
	}
}

// WithSessionReleaseMsgFunc set a func to release message once the message encode into the write buf
func WithSessionReleaseMsgFunc(value func(any)) Option {
	return func(bio *baseIO) {
		bio.options.releaseMsgFunc = value
	}
}

// IOSession internally holds a raw net.Conn on which to provide read and write operations
type IOSession interface {
	// ID session id
	ID() uint64
	// Connect connect to address, only used at client-side
	Connect(addr string, timeout time.Duration) error
	// Connected returns true if connection is ok
	Connected() bool
	// Disconnect disconnect the connection
	Disconnect() error
	// Close close the session, the read and write buffer will closed, and cannot Connect
	// again. IOSession reference count minus 1.
	Close() error
	// Ref for IOSessions, held by several goroutines, several references are needed. Each
	// concurrent process holding an IOSession can Close the IOSession and release the resource
	// when the reference count reaches 0.
	Ref()
	// Read read packet from connection
	Read(option ReadOptions) (any, error)
	// Write encodes the msg into a []byte into the buffer according to the codec.Encode.
	// If flush is set to flase, the data will not be written to the underlying socket.
	Write(msg any, options WriteOptions) error
	// Flush flush the out buffer
	Flush(timeout time.Duration) error
	// RemoteAddress returns remote address, include ip and port
	RemoteAddress() string
}

type baseIO struct {
	id                    uint64
	state                 int32
	conn                  net.Conn
	localAddr, remoteAddr string
	in                    *buf.ByteBuf
	out                   *buf.ByteBuf
	disableConnect        bool
	logger                *zap.Logger
	readCopyBuf           []byte
	writeCopyBuf          []byte

	options struct {
		aware                             IOSessionAware
		codec                             codec.Codec
		readBufSize, writeBufSize         int
		readCopyBufSize, writeCopyBufSize int
		connOptionFunc                    func(net.Conn)
		releaseMsgFunc                    func(any)
		allocator                         buf.Allocator
	}

	atomic struct {
		ref int32
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
	if bio.options.aware != nil {
		bio.options.aware.Created(bio)
	}
	bio.Ref()
	return bio
}

func (bio *baseIO) adjust() {
	bio.logger = adjustLogger(bio.logger)
	if bio.options.readBufSize == 0 {
		bio.options.readBufSize = defaultReadBuf
	}
	if bio.options.readCopyBufSize == 0 {
		bio.options.readCopyBufSize = defaultReadCopyBuf
	}
	if bio.options.writeBufSize == 0 {
		bio.options.writeBufSize = defaultWriteBuf
	}
	if bio.options.writeCopyBufSize == 0 {
		bio.options.writeCopyBufSize = defaultWriteCopyBuf
	}
	if bio.options.releaseMsgFunc == nil {
		bio.options.releaseMsgFunc = func(any) {}
	}
	if bio.options.connOptionFunc == nil {
		bio.options.connOptionFunc = func(net.Conn) {}
	}
}

func (bio *baseIO) ID() uint64 {
	return bio.id
}

func (bio *baseIO) Connect(addressWithNetwork string, timeout time.Duration) error {
	network, address, err := parseAdddress(addressWithNetwork)
	if err != nil {
		return err
	}

	if bio.disableConnect {
		return ErrDisableConnect
	}

	old := bio.getState()
	switch old {
	case stateReadyToConnect:
		break
	case stateClosed:
		return fmt.Errorf("the session is closed")
	case stateConnecting:
		return fmt.Errorf("the session is connecting in other goroutine")
	case stateConnected:
		return nil
	}

	if !atomic.CompareAndSwapInt32(&bio.state, stateReadyToConnect, stateConnecting) {
		current := bio.getState()
		if current == stateConnected {
			return nil
		}
		return fmt.Errorf("the session is closing or connecting is other goroutine")
	}

	d := net.Dialer{Timeout: timeout, Control: nil}
	conn, err := d.Dial(network, address)
	if nil != err {
		atomic.StoreInt32(&bio.state, stateReadyToConnect)
		return err
	}

	bio.conn = conn
	bio.initConn()
	return nil
}

func (bio *baseIO) Connected() bool {
	return bio.getState() == stateConnected
}

func (bio *baseIO) Disconnect() error {
	old := bio.getState()
	switch old {
	case stateReadyToConnect, stateClosed:
		return nil
	case stateConnecting:
		return fmt.Errorf("the session is connecting in other goroutine")
	case stateConnected:
		break
	}

	if !atomic.CompareAndSwapInt32(&bio.state, stateConnected, stateReadyToConnect) {
		current := bio.getState()
		if current == stateReadyToConnect {
			return nil
		}
		return fmt.Errorf("the session is closing or connecting is other goroutine")
	}

	bio.closeConn()
	atomic.StoreInt32(&bio.state, stateReadyToConnect)
	return nil
}

func (bio *baseIO) Ref() {
	atomic.AddInt32(&bio.atomic.ref, 1)
}

func (bio *baseIO) unRef() int32 {
	return atomic.AddInt32(&bio.atomic.ref, -1)
}

func (bio *baseIO) Close() error {
	ref := bio.unRef()
	if ref < 0 {
		panic("invalid ref count")
	}
	if ref > 0 {
		return nil
	}

	old := bio.getState()
	switch old {
	case stateReadyToConnect, stateClosed:
		return nil
	case stateConnecting:
		return fmt.Errorf("the session is connecting in other goroutine")
	case stateConnected:
		break
	}

	if !atomic.CompareAndSwapInt32(&bio.state, stateConnected, stateClosed) {
		current := bio.getState()
		if current == stateClosed {
			return nil
		}
		return fmt.Errorf("the session is closing or connecting is other goroutine")
	}

	bio.closeConn()
	bio.out.Close()
	bio.in.Close()
	atomic.StoreInt32(&bio.state, stateClosed)
	if bio.options.aware != nil {
		bio.options.aware.Closed(bio)
	}
	bio.logger.Debug("IOSession closed")
	return nil
}

func (bio *baseIO) Read(options ReadOptions) (any, error) {
	for {
		if !bio.Connected() {
			return nil, ErrIllegalState
		}

		var msg any
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
				bio.in.Reset()
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

func (bio *baseIO) Write(msg any, options WriteOptions) error {
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

func (bio *baseIO) RemoteAddress() string {
	return bio.remoteAddr
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
		if err := bio.conn.Close(); err != nil {
			bio.logger.Error("close conneciton failed",
				zap.Error(err))
			return
		}
		bio.logger.Error("conneciton disconnected")
	}
}

func (bio *baseIO) getState() int32 {
	return atomic.LoadInt32(&bio.state)
}

func (bio *baseIO) initConn() {
	if bio.options.connOptionFunc != nil {
		bio.options.connOptionFunc(bio.conn)
	}
	bio.remoteAddr = bio.conn.RemoteAddr().String()
	bio.localAddr = bio.conn.LocalAddr().String()
	bio.in = buf.NewByteBuf(bio.options.readBufSize, buf.WithMemAllocator(bio.options.allocator))
	bio.out = buf.NewByteBuf(bio.options.writeBufSize, buf.WithMemAllocator(bio.options.allocator))
	bio.logger = adjustLogger(bio.logger).Named("io-session").With(zap.Uint64("id", bio.id),
		zap.String("local-address", bio.localAddr),
		zap.String("remote-address", bio.remoteAddr))
	atomic.StoreInt32(&bio.state, stateConnected)
	bio.logger.Debug("session init completed")
}
