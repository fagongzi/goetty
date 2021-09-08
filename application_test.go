package goetty

import (
	"testing"
	"time"

	"github.com/fagongzi/goetty/codec/simple"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

var (
	testAddr = "127.0.0.1:12345"
)

func TestStart(t *testing.T) {
	app := newTestTCPApp(t, nil)
	defer app.Stop()

	assert.NoError(t, app.Start())
	assert.NoError(t, app.Start())
}

func TestStop(t *testing.T) {
	app := newTestTCPApp(t, nil).(*server)
	assert.NoError(t, app.Start())
	n := 200
	for i := 0; i < n; i++ {
		session := newTestIOSession(t)
		ok, err := session.Connect(testAddr, time.Second)
		assert.NoError(t, err)
		assert.True(t, ok)
	}
	time.Sleep(time.Second * 1)

	var sessions []IOSession
	for _, m := range app.sessions {
		for _, s := range m.sessions {
			sessions = append(sessions, s)
		}
	}

	assert.Equal(t, n, len(sessions))
	assert.NoError(t, app.Stop())
}

func TestCloseBlock(t *testing.T) {
	app := newTestTCPApp(t, nil).(*server)
	assert.NoError(t, app.Start())

	conn := newTestIOSession(t, WithEnableAsyncWrite(16), WithLogger(zap.NewExample()))
	ok, err := conn.Connect(testAddr, time.Second)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.NoError(t, app.Stop())
	assert.NoError(t, conn.Write(string(make([]byte, 1024*1024))))
	assert.NoError(t, conn.Close())
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
