package simple

import (
	"testing"

	"github.com/fagongzi/goetty"
	"github.com/stretchr/testify/assert"
)

func TestBytesCodec(t *testing.T) {
	v := []byte("hello world")
	buf := goetty.NewByteBuf(32)

	decoder, encoder := NewBytesCodec()

	assert.NoError(t, encoder.Encode(v, buf), "TestBytesCodec failed")

	completed, readed, err := decoder.Decode(buf)
	assert.NoError(t, err, "TestBytesCodec failed")
	assert.True(t, completed, "TestBytesCodec failed")
	assert.Equal(t, v, readed, "TestBytesCodec failed")
}
