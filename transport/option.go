package transport

import (
	"github.com/fagongzi/goetty"
)

// Option transport option
type Option func(*options)

type options struct {
	logger               Logger
	factory              func(size int64) Queue
	respReleaseFunc      func(interface{})
	errorResponseFactory func(interface{}, error) interface{}
	decoder              goetty.Decoder
	encoder              goetty.Encoder
}

func (opts *options) adjust() {
	if opts.logger == nil {
		opts.logger = newStdLog()
	}

	if opts.decoder == nil || opts.encoder == nil {
		opts.logger.Fatalf("decoder or encoder not set, use WithCodec option")
	}

	if opts.factory == nil {
		opts.factory = func(size int64) Queue {
			return newQueue(size)
		}
	}
}

// WithLogger set logger
func WithLogger(value Logger) Option {
	return func(opts *options) {
		opts.logger = value
	}
}

// WithQueueFactory set queue factory
func WithQueueFactory(value func(size int64) Queue) Option {
	return func(opts *options) {
		opts.factory = value
	}
}

// WithRespReleaseFunc set queue factory
func WithRespReleaseFunc(value func(interface{})) Option {
	return func(opts *options) {
		opts.respReleaseFunc = value
	}
}

// WithCodec set codec
func WithCodec(decoder goetty.Decoder, encoder goetty.Encoder) Option {
	return func(opts *options) {
		opts.decoder = decoder
		opts.encoder = encoder
	}
}

// WithErrorResponseFactory set error response factory
func WithErrorResponseFactory(value func(interface{}, error) interface{}) Option {
	return func(opts *options) {
		opts.errorResponseFactory = value
	}
}
