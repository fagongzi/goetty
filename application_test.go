package goetty

import (
	"os"
	"sync"
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
			app := newTestApp(t, addr, func(i IOSession, a any, u uint64) error {
				return i.Write(a, WriteOptions{Flush: true})
			}).(*server)
			assert.NoError(t, app.Start())

			n := 10
			wg := &sync.WaitGroup{}
			for i := 0; i < n; i++ {
				session := newTestIOSession(t)
				err := session.Connect(addr, time.Second)
				assert.NoError(t, err)
				wg.Add(1)
				go func(conn IOSession) {
					for {
						_, err := conn.Read(ReadOptions{})
						if err != nil {
							wg.Done()
							return
						}
					}
				}(session)
			}

			assert.NoError(t, app.Stop())
			wg.Wait()
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
	handleFunc func(IOSession, any, uint64) error,
	opts ...AppOption) NetApplication {
	return newTestAppWithCodec(t, address, handleFunc, simple.NewStringCodec())
}

func newTestAppWithCodec(t assert.TestingT,
	address string,
	handleFunc func(IOSession, any, uint64) error,
	codec codec.Codec,
	opts ...AppOption) NetApplication {
	if handleFunc == nil {
		handleFunc = func(i1 IOSession, i2 any, u uint64) error {
			return nil
		}
	}
	assert.NoError(t, os.RemoveAll(testUnixSocket[7:]))
	opts = append(opts, WithAppSessionOptions(WithSessionCodec(codec)))
	app, err := NewApplication(address, handleFunc, opts...)
	assert.NoError(t, err)
	return app
}

func newTestIOSession(t *testing.T, opts ...Option) IOSession {
	opts = append([]Option{WithSessionCodec(simple.NewStringCodec())}, opts...)
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
