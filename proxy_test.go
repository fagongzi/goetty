package goetty

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	proxyAddress     = "unix:///tmp/proxy.sock"
	upstream1Address = "unix:///tmp/upstream-1.sock"
	upstream2Address = "unix:///tmp/upstream-2.sock"
)

func TestProxy(t *testing.T) {
	assert.NoError(t, os.RemoveAll(proxyAddress[7:]))
	proxy := NewProxy(proxyAddress, nil)
	assert.NoError(t, proxy.Start())
	defer func() {
		assert.NoError(t, proxy.Stop())
	}()

	upstream1 := newTestApp(t, upstream1Address, func(i IOSession, a any, u uint64) error {
		return i.Write("upstream1", WriteOptions{Flush: true})
	})
	assert.NoError(t, upstream1.Start())
	defer func() {
		assert.NoError(t, upstream1.Stop())
	}()

	upstream2 := newTestApp(t, upstream2Address, func(i IOSession, a any, u uint64) error {
		return i.Write("upstream2", WriteOptions{Flush: true})
	})
	assert.NoError(t, upstream2.Start())
	defer func() {
		assert.NoError(t, upstream2.Stop())
	}()

	proxy.AddUpStream(upstream1Address, time.Second)
	proxy.AddUpStream(upstream2Address, time.Second)

	c1 := newTestIOSession(t)
	defer func() {
		assert.NoError(t, c1.Close())
	}()

	assert.NoError(t, c1.Connect(proxyAddress, time.Second))
	assert.NoError(t, c1.Write("test", WriteOptions{Flush: true}))
	v, err := c1.Read(ReadOptions{})
	assert.NoError(t, err)
	assert.Equal(t, "upstream1", v)

	c2 := newTestIOSession(t)
	defer func() {
		assert.NoError(t, c2.Close())
	}()
	assert.NoError(t, c2.Connect(proxyAddress, time.Second))
	assert.NoError(t, c2.Write("test", WriteOptions{Flush: true}))
	v, err = c2.Read(ReadOptions{})
	assert.NoError(t, err)
	assert.Equal(t, "upstream2", v)
}
