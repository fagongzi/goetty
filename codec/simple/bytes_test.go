package simple

import (
	"testing"

	"github.com/fagongzi/goetty/v3/buf"
	"github.com/stretchr/testify/assert"
)

func TestBytesCodec(t *testing.T) {
	v := []byte("hello world")
	buf := buf.NewByteBuf(32)
	codec := NewBytesCodec()
	assert.NoError(t, codec.Encode(v, buf, nil), "TestBytesCodec failed")
	readed, completed, err := codec.Decode(buf)
	assert.NoError(t, err, "TestBytesCodec failed")
	assert.True(t, completed, "TestBytesCodec failed")
	assert.Equal(t, v, readed, "TestBytesCodec failed")
}
