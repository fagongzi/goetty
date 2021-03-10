package simple

import (
	"testing"

	"github.com/fagongzi/goetty/buf"
	"github.com/stretchr/testify/assert"
)

func TestStringCodec(t *testing.T) {
	v := "hello world"
	buf := buf.NewByteBuf(32)

	encoder, decoder := NewStringCodec()

	assert.NoError(t, encoder.Encode(v, buf), "TestStringCodec failed")

	completed, readed, err := decoder.Decode(buf)
	assert.NoError(t, err, "TestStringCodec failed")
	assert.True(t, completed, "TestStringCodec failed")
	assert.Equal(t, v, readed, "TestStringCodec failed")
}
