package simple

import (
	"testing"

	"github.com/fagongzi/goetty"
	"github.com/stretchr/testify/assert"
)

func TestStringCodec(t *testing.T) {
	v := "hello world"
	buf := goetty.NewByteBuf(32)

	decoder, encoder := NewStringCodec()

	assert.NoError(t, encoder.Encode(v, buf), "TestStringCodec failed")

	completed, readed, err := decoder.Decode(buf)
	assert.NoError(t, err, "TestStringCodec failed")
	assert.True(t, completed, "TestStringCodec failed")
	assert.Equal(t, v, readed, "TestStringCodec failed")
}
