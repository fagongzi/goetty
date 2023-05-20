package length

import (
	"io"
	"testing"

	"github.com/fagongzi/goetty/v2/buf"
	"github.com/stretchr/testify/assert"
)

func TestEncode(t *testing.T) {
	baseCodec := &bytesCodec{}
	codec := New(baseCodec)
	buf := buf.NewByteBuf(32)
	err := codec.Encode([]byte("hello"), buf, nil)
	assert.NoError(t, err)

	err = codec.Encode([]byte("world"), buf, nil)
	assert.NoError(t, err)

	assert.Equal(t, 18, buf.Readable())

	n := buf.ReadInt()
	assert.Equal(t, 5, n)

	_, v := buf.ReadBytes(n)
	assert.Equal(t, "hello", string(v))

	n = buf.ReadInt()
	assert.Equal(t, 5, n)

	_, v = buf.ReadBytes(n)
	assert.Equal(t, "world", string(v))
}

func TestDecodeWithZeroLength(t *testing.T) {
	baseCodec := &bytesCodec{}
	codec := New(baseCodec)
	buf := buf.NewByteBuf(32)
	buf.WriteInt(0)
	_, _, err := codec.Decode(buf)
	assert.Error(t, err)
}

type bytesCodec struct {
}

func (c *bytesCodec) Decode(in *buf.ByteBuf) (any, bool, error) {
	return in.ReadMarkedData(), true, nil
}

func (c *bytesCodec) Encode(data any, out *buf.ByteBuf, conn io.Writer) error {
	bytes, _ := data.([]byte)
	out.Write(bytes)
	return nil
}
