package goetty

import (
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fagongzi/goetty/v2/buf"
	"github.com/fagongzi/goetty/v2/codec"
	"github.com/fagongzi/goetty/v2/codec/length"
	"github.com/lni/goutils/leaktest"
	"github.com/stretchr/testify/assert"
)

func TestNormal(t *testing.T) {
	defer leaktest.AfterTest(t)()

	for name, address := range testAddresses {
		addr := address
		t.Run(name, func(t *testing.T) {
			cnt := uint64(0)
			app := newTestApp(t, addr, func(rs IOSession, msg any, received uint64) error {
				atomic.StoreUint64(&cnt, received)
				assert.NoError(t, rs.Write(msg, WriteOptions{Flush: true}))
				return nil
			})
			app.Start()
			defer app.Stop()

			client := newTestIOSession(t)
			err := client.Connect(addr, time.Second)
			assert.NoError(t, err)
			assert.True(t, client.Connected())

			assert.NoError(t, client.Write("hello", WriteOptions{Flush: true}))
			reply, err := client.Read(ReadOptions{})
			assert.NoError(t, err)
			assert.Equal(t, "hello", reply)
			assert.Equal(t, uint64(1), atomic.LoadUint64(&cnt))

			assert.NoError(t, client.Disconnect())
			assert.False(t, client.Connected())
			assert.Error(t, client.Write("hello", WriteOptions{Flush: true}))

			err = client.Connect(addr, time.Second)
			assert.NoError(t, err)
			assert.True(t, client.Connected())
		})
	}
}

func TestTLSNormal(t *testing.T) {
	defer leaktest.AfterTest(t)()

	for name, address := range testAddresses {
		addr := address
		t.Run(name, func(t *testing.T) {
			app := newTestApp(t, addr, func(rs IOSession, msg any, received uint64) error {
				assert.NoError(t, rs.Write(msg, WriteOptions{Flush: true}))
				return nil
			}, WithAppTLSFromCertAndKey(
				"./etc/server-cert.pem",
				"./etc/server-key.pem",
				"./etc/ca.pem",
				true))
			app.Start()
			defer app.Stop()

			client := newTestIOSession(t,
				WithSessionTLSFromCertAndKeys(
					"./etc/client-cert.pem",
					"./etc/client-key.pem",
					"./etc/ca.pem",
					true),
			)
			err := client.Connect(addr, time.Second*5)
			assert.NoError(t, err)
			assert.True(t, client.Connected())

			assert.NoError(t, client.Write("hello", WriteOptions{Flush: true}))
			reply, err := client.Read(ReadOptions{})
			assert.NoError(t, err)
			assert.Equal(t, "hello", reply)
		})
	}
}

func TestReadWithTimeout(t *testing.T) {
	defer leaktest.AfterTest(t)()

	for name, address := range testAddresses {
		addr := address
		t.Run(name, func(t *testing.T) {
			app := newTestApp(t, addr, func(rs IOSession, msg any, received uint64) error {
				rs.Write(msg, WriteOptions{Flush: true})
				return nil
			})
			app.Start()
			defer app.Stop()

			client := newTestIOSession(t)
			defer client.Close()

			err := client.Connect(addr, time.Second)
			assert.NoError(t, err)

			_, err = client.Read(ReadOptions{Timeout: time.Millisecond * 10})
			assert.Error(t, err)
		})
	}
}

func TestWriteWithTimeout(t *testing.T) {
	defer leaktest.AfterTest(t)()

	for name, address := range testAddresses {
		addr := address
		t.Run(name, func(t *testing.T) {
			app := newTestApp(t, addr, func(rs IOSession, msg any, received uint64) error {
				rs.Write(msg, WriteOptions{Flush: true})
				return nil
			})
			app.Start()
			defer func() {
				assert.NoError(t, app.Stop())
			}()

			client := newTestIOSession(t)
			defer client.Close()

			err := client.Connect(addr, time.Second)
			assert.NoError(t, err)

			err = client.Write("hello", WriteOptions{Flush: true, Timeout: 1})
			assert.Error(t, err)
		})
	}
}

func BenchmarkWriteAndRead(b *testing.B) {
	b.ReportAllocs()
	codec := newBenchmarkStringCodec()
	app := newTestAppWithCodec(b, testUnixSocket, func(rs IOSession, msg any, received uint64) error {
		rs.Write(msg, WriteOptions{Flush: true})
		return nil
	}, codec)
	app.Start()
	defer app.Stop()

	client := newTestIOSession(nil, WithSessionCodec(codec))
	defer client.Close()

	err := client.Connect(testUnixSocket, time.Second)
	assert.NoError(b, err)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := client.Write("ok", WriteOptions{Flush: true})
		assert.NoError(b, err)
		_, err = client.Read(ReadOptions{})
		assert.NoError(b, err)
	}
}

func newBenchmarkStringCodec() codec.Codec {
	return length.New(&stringCodec{})
}

type stringCodec struct {
}

func (c *stringCodec) Decode(in *buf.ByteBuf) (any, bool, error) {
	in.Skip(in.GetMarkedDataLen())
	return "OK", true, nil
}

func (c *stringCodec) Encode(data any, out *buf.ByteBuf, conn io.Writer) error {
	msg, _ := data.(string)
	for _, d := range msg {
		out.WriteByte(byte(d))
	}
	return nil
}
