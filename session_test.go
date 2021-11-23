package goetty

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/lni/goutils/leaktest"
	"github.com/stretchr/testify/assert"
)

func TestNormal(t *testing.T) {
	defer leaktest.AfterTest(t)

	var cs IOSession
	cnt := uint64(0)
	app := newTestTCPApp(t, func(rs IOSession, msg interface{}, received uint64) error {
		cs = rs
		rs.WriteAndFlush(msg)
		atomic.StoreUint64(&cnt, received)
		return nil
	})
	app.Start()
	defer app.Stop()

	client := newTestIOSession(t, WithTimeout(time.Second, time.Second))
	ok, err := client.Connect(testAddr, time.Second)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.True(t, client.Connected())

	assert.NoError(t, client.WriteAndFlush("hello"))
	reply, err := client.Read()
	assert.NoError(t, err)
	assert.Equal(t, "hello", reply)
	assert.Equal(t, uint64(1), atomic.LoadUint64(&cnt))

	v, err := app.GetSession(cs.ID())
	assert.NoError(t, err)
	assert.NotNil(t, v)

	assert.NoError(t, app.Broadcast("world"))
	reply, err = client.Read()
	assert.NoError(t, err)
	assert.Equal(t, "world", reply)

	assert.NoError(t, client.Close())
	assert.False(t, client.Connected())
	assert.Error(t, client.WriteAndFlush("hello"))

	time.Sleep(time.Millisecond * 100)
	v, err = app.GetSession(cs.ID())
	assert.NoError(t, err)
	assert.Nil(t, v)

	ok, err = client.Connect(testAddr, time.Second)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.True(t, client.Connected())
}

func TestAsyncWrite(t *testing.T) {
	defer leaktest.AfterTest(t)

	app := newTestTCPApp(t, func(rs IOSession, msg interface{}, received uint64) error {
		rs.WriteAndFlush(msg)
		return nil
	})
	app.Start()
	defer app.Stop()

	client := newTestIOSession(t, WithTimeout(time.Second, time.Second), WithEnableAsyncWrite(16))
	ok, err := client.Connect(testAddr, time.Second)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.True(t, client.Connected())

	assert.NoError(t, client.WriteAndFlush("hello"))
	reply, err := client.Read()
	assert.NoError(t, err)
	assert.Equal(t, "hello", reply)

	assert.NoError(t, client.Close())
	ok, err = client.Connect(testAddr, time.Second)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.True(t, client.Connected())
	assert.NoError(t, client.WriteAndFlush("hello"))
	reply, err = client.Read()
	assert.NoError(t, err)
	assert.Equal(t, "hello", reply)
}
