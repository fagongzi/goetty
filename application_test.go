package goetty

import (
	"testing"

	"github.com/fagongzi/goetty/codec/simple"
	"github.com/stretchr/testify/assert"
)

var (
	testAddr = "127.0.0.1:12345"
)

func TestStart(t *testing.T) {
	app := newTestTCPApp(t, nil)
	assert.NoError(t, app.Start())
	assert.NoError(t, app.Start())
	assert.NoError(t, app.Stop())
	assert.NoError(t, app.Stop())
}

func newTestTCPApp(t *testing.T, handleFunc func(IOSession, interface{}, uint64) error, opts ...AppOption) NetApplication {
	encoder, decoder := simple.NewStringCodec()
	opts = append(opts, WithAppSessionOptions(WithCodec(encoder, decoder)))
	app, err := NewTCPApplication(testAddr, handleFunc, opts...)
	assert.NoError(t, err)

	return app
}

func newTestIOSession(t *testing.T, opts ...Option) IOSession {
	encoder, decoder := simple.NewStringCodec()
	opts = append(opts, WithCodec(encoder, decoder))
	return NewIOSession(opts...)
}
