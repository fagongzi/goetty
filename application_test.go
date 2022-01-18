package goetty

import (
	"testing"
	"time"

	"github.com/fagongzi/goetty/codec/simple"
	"github.com/lni/goutils/leaktest"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

var (
	testAddr       = "127.0.0.1:12345"
	testUDPAddr    = "udp://127.0.0.1:12346"
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
				assert.NoError(t, session.WriteAndFlush("test"))
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

func TestCloseBlock(t *testing.T) {
	defer leaktest.AfterTest(t)()

	for name, address := range testAddresses {
		addr := address
		t.Run(name, func(t *testing.T) {
			app := newTestApp(t, addr, nil).(*server)
			assert.NoError(t, app.Start())

			conn := newTestIOSession(t, WithEnableAsyncWrite(16), WithLogger(zap.NewExample()))
			ok, err := conn.Connect(addr, time.Second)
			assert.NoError(t, err)
			assert.True(t, ok)
			assert.NoError(t, app.Stop())
			assert.NoError(t, conn.Write(string(make([]byte, 1024*1024))))
			assert.NoError(t, conn.Close())
		})
	}

}

func TestIssue13(t *testing.T) {
	defer leaktest.AfterTest(t)()

	for name, address := range testAddresses {
		addr := address
		t.Run(name, func(t *testing.T) {
			app := newTestApp(t, addr, nil).(*server)
			assert.NoError(t, app.Start())

			conn := newTestIOSession(t, WithEnableAsyncWrite(16), WithLogger(zap.NewExample()))
			ok, err := conn.Connect(addr, time.Second)
			assert.NoError(t, err)
			assert.True(t, ok)

			defer conn.Close()

			errC := make(chan error)
			go func() {
				_, err := conn.Read()
				if err != nil {
					errC <- err
					return
				}
			}()

			time.Sleep(time.Millisecond * 100)
			assert.NoError(t, app.Stop())

			select {
			case <-errC:
				return
			case <-time.After(time.Second * 1):
				assert.Fail(t, "timeout")
			}
		})
	}

}

func newTestApp(t *testing.T, address string, handleFunc func(IOSession, interface{}, uint64) error, opts ...AppOption) NetApplication {
	if handleFunc == nil {
		handleFunc = func(i1 IOSession, i2 interface{}, u uint64) error {
			return nil
		}
	}

	encoder, decoder := simple.NewStringCodec()
	opts = append(opts, WithAppSessionOptions(WithCodec(encoder, decoder)))
	app, err := NewApplication(address, handleFunc, opts...)
	assert.NoError(t, err)

	return app
}

func newTestIOSession(t *testing.T, opts ...Option) IOSession {
	encoder, decoder := simple.NewStringCodec()
	opts = append(opts, WithCodec(encoder, decoder))
	return NewIOSession(opts...)
}

func TestParseAddress(t *testing.T) {
	network, address, err := parseAdddress(testAddr)
	assert.NoError(t, err)
	assert.Equal(t, "tcp4", network)
	assert.Equal(t, testAddr, address)

	network, address, err = parseAdddress(testUDPAddr)
	assert.NoError(t, err)
	assert.Equal(t, "udp", network)
	assert.Equal(t, "127.0.0.1:12346", address)

	network, address, err = parseAdddress(testUnixSocket)
	assert.NoError(t, err)
	assert.Equal(t, "unix", network)
	assert.Equal(t, "/tmp/goetty.sock", address)
}
