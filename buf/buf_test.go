package buf

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewWrap(t *testing.T) {
	value := []byte{1, 2, 3}
	buf := WrapBytes(value)
	assert.Equal(t, 3, buf.Readable())

	v, err := buf.ReadByte()
	assert.NoError(t, err)
	assert.Equal(t, byte(1), v)

	v, err = buf.ReadByte()
	assert.NoError(t, err)
	assert.Equal(t, byte(2), v)

	v, err = buf.ReadByte()
	assert.NoError(t, err)
	assert.Equal(t, byte(3), v)
}

func TestWrap(t *testing.T) {
	buf := NewByteBuf(4)
	_, err := buf.Write([]byte{5, 6, 7})
	assert.NoError(t, err)

	value := []byte{1, 2, 3}
	buf.Wrap(value)
	assert.Equal(t, 3, buf.Readable())

	v, err := buf.ReadByte()
	assert.NoError(t, err)
	assert.Equal(t, byte(1), v)

	v, err = buf.ReadByte()
	assert.NoError(t, err)
	assert.Equal(t, byte(2), v)

	v, err = buf.ReadByte()
	assert.NoError(t, err)
	assert.Equal(t, byte(3), v)
}

func TestSlice(t *testing.T) {
	buf := NewByteBuf(32)
	_, err := buf.Write([]byte("hello"))
	assert.NoError(t, err)
	s := buf.Slice(0, 5)
	assert.Equal(t, "hello", string(s.Data()))
}

func TestWrittenDataAfterMark(t *testing.T) {
	buf := NewByteBuf(32)
	buf.MarkWrite()
	_, err := buf.Write([]byte("hello"))
	assert.NoError(t, err)
	s := buf.WrittenDataAfterMark()
	assert.Equal(t, "hello", string(s.Data()))
}

func TestExpansion(t *testing.T) {
	buf := NewByteBuf(256)
	data := make([]byte, 257)
	_, err := buf.Write(data)
	assert.NoError(t, err)
	assert.Equal(t, 512, cap(buf.buf))
}

func TestWriteToSink(t *testing.T) {
	buf := NewByteBuf(2)
	buf2 := NewByteBuf(2)

	buf.SetSinkTo(buf2)
	n, err := buf.WriteToSink([]byte("hello"), 2)
	assert.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, []byte("hello"), buf2.ReadableBytes())

	n, err = buf.WriteToSink([]byte("hello"), 0)
	assert.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, []byte("hello"), buf2.ReadableBytes()[5:])

	n, err = buf.WriteToSink([]byte("hello"), 5)
	assert.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, []byte("hello"), buf2.ReadableBytes()[10:])
}

func TestFlushToSink(t *testing.T) {
	buf := NewByteBuf(2)
	buf2 := NewByteBuf(2)

	buf.SetSinkTo(buf2)

	MustWriteString(buf, "hello")
	MustFlushToSink(buf)

	assert.Equal(t, []byte("hello"), buf2.ReadableBytes())
}

func BenchmarkWriteToSink(b *testing.B) {
	buf := NewByteBuf(1)
	buf2 := NewByteBuf(1)
	buf.SetSinkTo(buf2)
	v := []byte("hello000")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		MustWriteToSink(buf, v, 1)
		buf2.Clear()
	}
}
