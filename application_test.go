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
	testAddr            = "127.0.0.1:12345"
	testUnixSocket      = "unix:///tmp/goetty.sock"
	testListenAddresses = []string{testAddr, testUnixSocket}
	testAddresses       = map[string]string{
		"tcp":  testAddr,
		"unix": testUnixSocket,
	}
)

func TestStart(t *testing.T) {
	defer leaktest.AfterTest(t)()

	for name := range testAddresses {
		t.Run(name, func(t *testing.T) {
			app := newTestApp(t, testListenAddresses, nil)
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
			app := newTestApp(t,
				testListenAddresses,
				func(i IOSession[string, string], a string, u uint64) error {
					return i.Write(a, WriteOptions{Flush: true})
				}).(*server[string, string])
			assert.NoError(t, app.Start())

			n := 10
			wg := &sync.WaitGroup{}
			for i := 0; i < n; i++ {
				session := newTestIOSession(t)
				err := session.Connect(addr, time.Second)
				assert.NoError(t, err)
				wg.Add(1)
				go func(conn IOSession[string, string]) {
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

func TestStartWithTLS(t *testing.T) {
	defer leaktest.AfterTest(t)()

	for name := range testAddresses {
		t.Run(name, func(t *testing.T) {
			app := newTestApp(t,
				testListenAddresses,
				func(i IOSession[string, string], a string, u uint64) error {
					return i.Write(a, WriteOptions{Flush: true})
				},
				WithAppTLSFromCertAndKey[string, string](
					"./etc/server-cert.pem",
					"./etc/server-key.pem",
					"./etc/ca.pem",
					true))
			assert.NoError(t, app.Start())
			defer func() {
				assert.NoError(t, app.Stop())
			}()
		})
	}
}

func newTestApp(t assert.TestingT,
	addresses []string,
	handleFunc func(IOSession[string, string], string, uint64) error,
	opts ...AppOption[string, string]) NetApplication[string, string] {
	return newTestAppWithCodec(
		t,
		addresses,
		handleFunc,
		simple.NewStringCodec(),
		opts...)
}

func newTestAppWithCodec[IN string, OUT string](t assert.TestingT,
	addresses []string,
	handleFunc func(IOSession[string, string], string, uint64) error,
	codec codec.Codec[string, string],
	opts ...AppOption[string, string]) NetApplication[string, string] {
	if handleFunc == nil {
		handleFunc = func(i1 IOSession[string, string], i2 string, u uint64) error {
			return nil
		}
	}
	assert.NoError(t, os.RemoveAll(testUnixSocket[7:]))
	opts = append(opts, WithAppSessionOptions(WithSessionCodec(codec)))
	app, err := NewApplicationWithListenAddress(addresses, handleFunc, opts...)
	assert.NoError(t, err)
	return app
}

func newTestIOSession(t *testing.T, opts ...Option[string, string]) IOSession[string, string] {
	opts = append([]Option[string, string]{WithSessionCodec(simple.NewStringCodec())}, opts...)
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
