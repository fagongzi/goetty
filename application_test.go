package goetty

import (
	"testing"
	"time"

	"github.com/fagongzi/goetty/v2/codec"
	"github.com/fagongzi/goetty/v2/codec/simple"
	"github.com/lni/goutils/leaktest"
	"github.com/stretchr/testify/assert"
)

var (
	testAddr       = "127.0.0.1:12345"
	testUnixSocket = "unix:///tmp/goetty.sock"

	testAddresses = map[string]string{
		"tcp":  testAddr,
		"unix": testUnixSocket,
	}
)

func TestStart(t *testing.T) {
	defer leaktest.AfterTest(t)()

	for name, address := range testAddresses {
		addr := address
		t.Run(name, func(t *testing.T) {
			app := newTestApp(t, addr, nil)
			defer app.Stop()

			assert.NoError(t, app.Start())
			assert.NoError(t, app.Start())
		})
	}
}

func TestStop(t *testing.T) {
	defer leaktest.AfterTest(t)()

	for name, address := range testAddresses {
		addr := address
		t.Run(name, func(t *testing.T) {
			app := newTestApp(t, addr, nil).(*server)
			assert.NoError(t, app.Start())

			n := 10
			for i := 0; i < n; i++ {
				session := newTestIOSession(t)
				ok, err := session.Connect(addr, time.Second)
				assert.NoError(t, err)
				assert.True(t, ok)
				assert.NoError(t, session.Write("test", WriteOptions{Flush: true}))
			}

			assert.NoError(t, app.Stop())

			c := 0
			for _, m := range app.sessions {
				m.Lock()
				c += len(m.sessions)
				m.Unlock()
			}

			assert.Equal(t, 0, c)
		})
	}

}

func newTestApp(t assert.TestingT,
	address string,
	handleFunc func(IOSession, interface{}, uint64) error,
	opts ...AppOption) NetApplication {
	encoder, decoder := simple.NewStringCodec()
	return newTestAppWithCodec(t, address, handleFunc, encoder, decoder)
}

func newTestAppWithCodec(t assert.TestingT,
	address string,
	handleFunc func(IOSession, interface{}, uint64) error,
	encoder codec.Encoder,
	decoder codec.Decoder,
	opts ...AppOption) NetApplication {
	if handleFunc == nil {
		handleFunc = func(i1 IOSession, i2 interface{}, u uint64) error {
			return nil
		}
	}

	opts = append(opts, WithAppSessionOptions(WithCodec(encoder, decoder)))
	app, err := NewApplication(address, handleFunc, opts...)
	assert.NoError(t, err)
	return app
}

func newTestIOSession(t *testing.T, opts ...Option) IOSession {
	encoder, decoder := simple.NewStringCodec()
	return newTestIOSessionWithCodec(t, encoder, decoder, opts...)
}

func newTestIOSessionWithCodec(t *testing.T, encoder codec.Encoder, decoder codec.Decoder, opts ...Option) IOSession {
	opts = append(opts, WithCodec(encoder, decoder))
	return NewIOSession(opts...)
}

func TestParseAddress(t *testing.T) {
	network, address, err := parseAdddress(testAddr)
	assert.NoError(t, err)
	assert.Equal(t, "tcp4", network)
	assert.Equal(t, testAddr, address)

	network, address, err = parseAdddress(testUnixSocket)
	assert.NoError(t, err)
	assert.Equal(t, "unix", network)
	assert.Equal(t, "/tmp/goetty.sock", address)
}
