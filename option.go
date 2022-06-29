package goetty

import (
	"net"

	"github.com/fagongzi/goetty/v2/codec"
	"go.uber.org/zap"
)

const (
	// DefaultSessionBucketSize default bucket size of session map
	DefaultSessionBucketSize = uint64(64)
	// DefaultReadBuf read buf size
	DefaultReadBuf = 256
	// DefaultWriteBuf write buf size
	DefaultWriteBuf = 256
	// DefaultReadCopyBuf io.CopyBuffer buffer size for read
	DefaultReadCopyBuf = 1024
	// DefaultWriteCopyBuf io.CopyBuffer buffer size for write
	DefaultWriteCopyBuf = 1024
)

// IOSessionAware io session aware
type IOSessionAware interface {
	// Created session created
	Created(IOSession)
	//Closed session closed
	Closed(IOSession)
}

// AppOption application option
type AppOption func(*appOptions)

type appOptions struct {
	sessionOpts       *options
	sessionBucketSize uint64
	errorMsgFactory   func(IOSession, interface{}, error) interface{}
	aware             IOSessionAware
	logger            *zap.Logger
}

// WithAppSessionOptions set the number of maps to store session
func WithAppSessionOptions(value ...Option) AppOption {
	return func(opts *appOptions) {
		for _, opt := range value {
			opt(opts.sessionOpts)
		}
	}
}

// WithAppLogger set logger for application
func WithAppLogger(logger *zap.Logger) AppOption {
	return func(opts *appOptions) {
		opts.logger = logger
	}
}

// WithAppSessionBucketSize set the number of maps to store session
func WithAppSessionBucketSize(value uint64) AppOption {
	return func(opts *appOptions) {
		opts.sessionBucketSize = value
	}
}

// WithAppSessionBucketSize set the app session aware
func WithAppSessionAware(aware IOSessionAware) AppOption {
	return func(opts *appOptions) {
		opts.aware = aware
	}
}

// WithAppErrorMsgFactory set function to process error, closed the client session if this field not set
func WithAppErrorMsgFactory(value func(IOSession, interface{}, error) interface{}) AppOption {
	return func(opts *appOptions) {
		opts.errorMsgFactory = value
	}
}

func (opts *appOptions) adjust() {
	opts.logger = adjustLogger(opts.logger).Named("goetty")
	opts.sessionOpts.logger = opts.logger
	opts.sessionOpts.adjust()
	if opts.sessionBucketSize == 0 {
		opts.sessionBucketSize = DefaultSessionBucketSize
	}
}

// Option transport option
type Option func(*options)

type options struct {
	logger                            *zap.Logger
	decoder                           codec.Decoder
	encoder                           codec.Encoder
	readBufSize, writeBufSize         int
	readCopyBufSize, writeCopyBufSize int
	connOptionFunc                    func(net.Conn)
	releaseMsgFunc                    func(interface{})
	disableReleaseOut                 bool
}

func (opts *options) adjust() {
	if opts.readBufSize == 0 {
		opts.readBufSize = DefaultReadBuf
	}
	if opts.readCopyBufSize == 0 {
		opts.readCopyBufSize = DefaultReadCopyBuf
	}
	if opts.writeBufSize == 0 {
		opts.writeBufSize = DefaultWriteBuf
	}
	if opts.writeCopyBufSize == 0 {
		opts.writeCopyBufSize = DefaultWriteCopyBuf
	}
	if opts.releaseMsgFunc == nil {
		opts.releaseMsgFunc = func(interface{}) {}
	}
	if opts.connOptionFunc == nil {
		opts.connOptionFunc = func(net.Conn) {}
	}
	opts.logger = adjustLogger(opts.logger)
}

// WithDisableReleaseOutBuf set disable release buf
func WithDisableReleaseOutBuf() Option {
	return func(opts *options) {
		opts.disableReleaseOut = true
	}
}

// WithLogger set logger
func WithLogger(value *zap.Logger) Option {
	return func(opts *options) {
		opts.logger = value
	}
}

// WithConnOptionFunc set conn options func
func WithConnOptionFunc(connOptionFunc func(net.Conn)) Option {
	return func(opts *options) {
		opts.connOptionFunc = connOptionFunc
	}
}

// WithCodec set codec
func WithCodec(encoder codec.Encoder, decoder codec.Decoder) Option {
	return func(opts *options) {
		opts.encoder = encoder
		opts.decoder = decoder
	}
}

// WithBufSize set read/write buf size
func WithBufSize(read, write int) Option {
	return func(opts *options) {
		opts.readBufSize = read
		opts.writeBufSize = write
	}
}

// WithReleaseMsgFunc set the number of maps to store session
func WithReleaseMsgFunc(value func(interface{})) Option {
	return func(opts *options) {
		opts.releaseMsgFunc = value
	}
}
