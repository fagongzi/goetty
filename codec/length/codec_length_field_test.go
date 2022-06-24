package length

import (
	"testing"

	"github.com/fagongzi/goetty/v2/buf"
	"github.com/fagongzi/goetty/v2/codec/raw"
	"github.com/stretchr/testify/assert"
)

func TestEncode(t *testing.T) {
	baseEncoder, baseDecoder := raw.New()

	encoder, _ := New(baseEncoder, baseDecoder)
	buf := buf.NewByteBuf(32)
	err := encoder.Encode([]byte("hello"), buf)
	assert.NoError(t, err)

	err = encoder.Encode([]byte("world"), buf)
	assert.NoError(t, err)

	assert.Equal(t, 18, buf.Readable())

	n, _ := buf.ReadInt()
	assert.Equal(t, 5, n)

	_, v, _ := buf.ReadBytes(n)
	assert.Equal(t, "hello", string(v))

	n, _ = buf.ReadInt()
	assert.Equal(t, 5, n)

	_, v, _ = buf.ReadBytes(n)
	assert.Equal(t, "world", string(v))
}
