package goetty

import (
	"os"
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
			var cs IOSession
			cnt := uint64(0)
			app := newTestApp(t, addr, func(rs IOSession, msg interface{}, received uint64) error {
				cs = rs
				atomic.StoreUint64(&cnt, received)
				rs.Write(msg, WriteOptions{Flush: true})
				return nil
			})
			app.Start()
			defer app.Stop()

			client := newTestIOSession(t)
			ok, err := client.Connect(addr, time.Second)
			assert.NoError(t, err)
			assert.True(t, ok)
			assert.True(t, client.Connected())

			assert.NoError(t, client.Write("hello", WriteOptions{Flush: true}))
			reply, err := client.Read(ReadOptions{})
			assert.NoError(t, err)
			assert.Equal(t, "hello", reply)
			assert.Equal(t, uint64(1), atomic.LoadUint64(&cnt))

			v, err := app.GetSession(cs.ID())
			assert.NoError(t, err)
			assert.NotNil(t, v)

			assert.NoError(t, app.Broadcast("world"))
			reply, err = client.Read(ReadOptions{})
			assert.NoError(t, err)
			assert.Equal(t, "world", reply)

			assert.NoError(t, client.Close())
			assert.False(t, client.Connected())
			assert.Error(t, client.Write("hello", WriteOptions{Flush: true}))

			time.Sleep(time.Millisecond * 100)
			v, err = app.GetSession(cs.ID())
			assert.NoError(t, err)
			assert.Nil(t, v)

			ok, err = client.Connect(addr, time.Second)
			assert.NoError(t, err)
			assert.True(t, ok)
			assert.True(t, client.Connected())
		})
	}
}

func TestReadWithTimeout(t *testing.T) {
	defer leaktest.AfterTest(t)()

	for name, address := range testAddresses {
		addr := address
		t.Run(name, func(t *testing.T) {
			app := newTestApp(t, addr, func(rs IOSession, msg interface{}, received uint64) error {
				rs.Write(msg, WriteOptions{Flush: true})
				return nil
			})
			app.Start()
			defer app.Stop()

			client := newTestIOSession(t)
			defer client.Close()

			ok, err := client.Connect(addr, time.Second)
			assert.NoError(t, err)
			assert.True(t, ok)

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
			app := newTestApp(t, addr, func(rs IOSession, msg interface{}, received uint64) error {
				rs.Write(msg, WriteOptions{Flush: true})
				return nil
			})
			app.Start()
			defer app.Stop()

			client := newTestIOSession(t)
			defer client.Close()

			ok, err := client.Connect(addr, time.Second)
			assert.NoError(t, err)
			assert.True(t, ok)

			err = client.Write("hello", WriteOptions{Flush: true, Timeout: 1})
			assert.Error(t, err)
		})
	}
}

func BenchmarkWriteAndRead(b *testing.B) {
	b.ReportAllocs()
	encoder, decoder := newBenchmarkStringCodec()
	assert.NoError(b, os.RemoveAll(testUnixSocket))
	app := newTestAppWithCodec(b, testUnixSocket, func(rs IOSession, msg interface{}, received uint64) error {
		rs.Write(msg, WriteOptions{Flush: true})
		return nil
	}, encoder, decoder)
	app.Start()
	defer app.Stop()

	client := newTestIOSessionWithCodec(nil, encoder, decoder)
	defer client.Close()

	ok, err := client.Connect(testUnixSocket, time.Second)
	assert.NoError(b, err)
	assert.True(b, ok)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.Write("ok", WriteOptions{Flush: true})
		client.Read(ReadOptions{})
	}
}

func newBenchmarkStringCodec() (codec.Encoder, codec.Decoder) {
	c := &stringCodec{}
	return length.New(c, c)
}

type stringCodec struct {
}

func (c stringCodec) Decode(in *buf.ByteBuf) (bool, interface{}, error) {
	in.MarkedBytesReaded()
	return true, "OK", nil
}

func (c stringCodec) Encode(data interface{}, out *buf.ByteBuf) error {
	msg, _ := data.(string)
	for _, d := range msg {
		out.WriteByte(byte(d))
	}
	return nil
}
