package goetty

import (
	"fmt"
	"io"
	"net"
	"strings"
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

func TestUseConn(t *testing.T) {
	defer leaktest.AfterTest(t)()

	for name, address := range testAddresses {
		addr := address
		t.Run(name, func(t *testing.T) {
			conns := map[string]net.Conn{}
			app := newTestApp(t, addr, func(rs IOSession, v any, received uint64) error {
				msg := v.(string)
				if msg == "regist" {
					conns[fmt.Sprintf("%d", rs.ID())] = rs.RawConn()
					assert.NoError(t, rs.Write(fmt.Sprintf("%d", rs.ID()), WriteOptions{Flush: true}))
					return nil
				} else if strings.HasPrefix(msg, "use:") {
					id := strings.Split(msg, ":")[1]
					rs.UseConn(conns[id])
					assert.NoError(t, rs.Write("OK", WriteOptions{Flush: true}))
					return nil
				}
				assert.NoError(t, rs.Write(msg, WriteOptions{Flush: true}))
				return nil
			})
			app.Start()
			defer app.Stop()

			c1 := newTestIOSession(t)
			assert.NoError(t, c1.Connect(addr, time.Second))
			assert.True(t, c1.Connected())
			assert.NoError(t, c1.Write("regist", WriteOptions{Flush: true}))
			id1, err := c1.Read(ReadOptions{})
			assert.NoError(t, err)
			assert.NotEmpty(t, id1)

			c2 := newTestIOSession(t)
			assert.NoError(t, c2.Connect(addr, time.Second))
			assert.True(t, c2.Connected())
			assert.NoError(t, c2.Write("regist", WriteOptions{Flush: true}))
			id2, err := c2.Read(ReadOptions{})
			assert.NoError(t, err)
			assert.NotEmpty(t, id2)

			assert.NoError(t, c1.Write(fmt.Sprintf("use:%s", id2), WriteOptions{Flush: true}))
			reply, err := c2.Read(ReadOptions{})
			assert.NoError(t, err)
			assert.Equal(t, "OK", reply)

			assert.NoError(t, c2.Write(fmt.Sprintf("use:%s", id1), WriteOptions{Flush: true}))
			reply, err = c1.Read(ReadOptions{})
			assert.NoError(t, err)
			assert.Equal(t, "OK", reply)
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

func TestCloseOnAwareCreated(t *testing.T) {
	defer leaktest.AfterTest(t)()
	s := NewIOSession(WithSessionAware(&testAware{}))
	assert.NoError(t, s.Close())
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

type testAware struct {
}

func (ta *testAware) Created(rs IOSession) {
	_ = rs.Close()
}

func (ta *testAware) Closed(rs IOSession) {

}
