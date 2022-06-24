package simple

import (
	"testing"

	"github.com/fagongzi/goetty/v2/buf"
	"github.com/stretchr/testify/assert"
)

func TestBytesCodec(t *testing.T) {
	v := []byte("hello world")
	buf := buf.NewByteBuf(32)

	encoder, decoder := NewBytesCodec()

	assert.NoError(t, encoder.Encode(v, buf), "TestBytesCodec failed")

	completed, readed, err := decoder.Decode(buf)
	assert.NoError(t, err, "TestBytesCodec failed")
	assert.True(t, completed, "TestBytesCodec failed")
	assert.Equal(t, v, readed, "TestBytesCodec failed")
}
