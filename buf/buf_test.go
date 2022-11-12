package buf

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadIndex(t *testing.T) {
	buf := NewByteBuf(32)
	buf.SetWriteIndex(6)
	buf.SetReadIndex(5)
	assert.Equal(t, 5, buf.GetReadIndex())
}

func TestWriteIndex(t *testing.T) {
	buf := NewByteBuf(32)
	buf.SetWriteIndex(5)
	assert.Equal(t, 5, buf.GetWriteIndex())
}

func TestMarkIndex(t *testing.T) {
	buf := NewByteBuf(32)
	buf.SetWriteIndex(6)
	buf.SetMarkIndex(5)
	assert.Equal(t, 5, buf.GetMarkIndex())
}

func TestClearMark(t *testing.T) {
	buf := NewByteBuf(32)
	buf.SetWriteIndex(6)
	buf.SetMarkIndex(5)
	buf.ClearMark()
	assert.Equal(t, 0, buf.GetMarkIndex())
}

func TestMarkedDataLen(t *testing.T) {
	buf := NewByteBuf(32)
	buf.SetWriteIndex(6)
	buf.SetMarkIndex(5)
	assert.Equal(t, 5, buf.GetMarkedDataLen())
}

func TestReset(t *testing.T) {
	buf := NewByteBuf(32)
	buf.SetWriteIndex(5)
	buf.SetReadIndex(2)
	buf.SetMarkIndex(4)
	buf.Reset()
	assert.Equal(t, 0, buf.GetReadIndex())
	assert.Equal(t, 0, buf.GetWriteIndex())
	assert.Equal(t, 0, buf.GetMarkIndex())
}

func TestSkip(t *testing.T) {
	buf := NewByteBuf(32)
	buf.SetWriteIndex(5)
	buf.SetReadIndex(2)
	buf.Skip(1)
	assert.Equal(t, 3, buf.GetReadIndex())
}

func TestSlice(t *testing.T) {
	buf := NewByteBuf(32)
	buf.WriteString("hello")
	s := buf.Slice(0, 5)
	assert.Equal(t, "hello", string(s.Data()))
}

func TestRawSlice(t *testing.T) {
	buf := NewByteBuf(32)
	buf.WriteString("hello")
	assert.Equal(t, "hello", string(buf.RawSlice(0, 5)))
}

func TestReadByte(t *testing.T) {
	buf := NewByteBuf(32)
	buf.WriteString("hello")
	assert.Equal(t, byte('h'), buf.MustReadByte())
	assert.Equal(t, 1, buf.GetReadIndex())
}

func TestReadBytes(t *testing.T) {
	buf := NewByteBuf(32)
	buf.WriteString("hello")
	n, v := buf.ReadBytes(10)
	assert.Equal(t, 5, n)
	assert.Equal(t, "hello", string(v))
	assert.Equal(t, 5, buf.GetReadIndex())
}

func TestReadMarkedData(t *testing.T) {
	buf := NewByteBuf(32)
	buf.WriteString("hello")
	buf.SetMarkIndex(buf.GetWriteIndex())
	assert.Equal(t, "hello", string(buf.ReadMarkedData()))
	assert.Equal(t, 0, buf.GetMarkIndex())
	assert.Equal(t, 5, buf.GetReadIndex())
}

func TestReadAll(t *testing.T) {
	buf := NewByteBuf(32)
	buf.WriteString("hello")
	n, v := buf.ReadAll()
	assert.Equal(t, 5, n)
	assert.Equal(t, "hello", string(v))
	assert.Equal(t, 5, buf.GetReadIndex())
}

func TestReadAndWriteInt(t *testing.T) {
	buf := NewByteBuf(32)
	buf.WriteInt(1)
	assert.Equal(t, 4, buf.GetWriteIndex())
	assert.Equal(t, 1, buf.ReadInt())
	assert.Equal(t, 4, buf.GetReadIndex())
}

func TestReadAndWriteUint16(t *testing.T) {
	buf := NewByteBuf(32)
	buf.WriteUint16(1)
	assert.Equal(t, 2, buf.GetWriteIndex())
	assert.Equal(t, uint16(1), buf.ReadUint16())
	assert.Equal(t, 2, buf.GetReadIndex())
}

func TestReadAndWriteUint32(t *testing.T) {
	buf := NewByteBuf(32)
	buf.WriteUint32(1)
	assert.Equal(t, 4, buf.GetWriteIndex())
	assert.Equal(t, uint32(1), buf.ReadUint32())
	assert.Equal(t, 4, buf.GetReadIndex())
}

func TestReadAndWriteUint64(t *testing.T) {
	buf := NewByteBuf(32)
	buf.WriteUint64(1)
	assert.Equal(t, 8, buf.GetWriteIndex())
	assert.Equal(t, uint64(1), buf.ReadUint64())
	assert.Equal(t, 8, buf.GetReadIndex())
}

func TestReadAndWriteInt64(t *testing.T) {
	buf := NewByteBuf(32)
	buf.WriteInt64(1)
	assert.Equal(t, 8, buf.GetWriteIndex())
	assert.Equal(t, int64(1), buf.ReadInt64())
	assert.Equal(t, 8, buf.GetReadIndex())
}

func TestMustWrite(t *testing.T) {
	buf := NewByteBuf(32)
	buf.MustWrite([]byte{1, 2, 3, 4, 5})
	assert.Equal(t, 5, buf.GetWriteIndex())
	n, v := buf.ReadAll()
	assert.Equal(t, 5, n)
	assert.Equal(t, []byte{1, 2, 3, 4, 5}, v)
}

func TestMustWriteByte(t *testing.T) {
	buf := NewByteBuf(32)
	buf.MustWriteByte(1)
	assert.Equal(t, 1, buf.GetWriteIndex())
	n, v := buf.ReadAll()
	assert.Equal(t, 1, n)
	assert.Equal(t, []byte{1}, v)
}

func TestWriteTo(t *testing.T) {
	buf := NewByteBuf(1)
	buf2 := NewByteBuf(1)
	buf.MustWrite([]byte{1, 2, 3, 4, 5})
	n, err := buf.WriteTo(buf2)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), n)
	_, v := buf2.ReadAll()
	assert.Equal(t, []byte{1, 2, 3, 4, 5}, v)
}

func TestReadFrom(t *testing.T) {
	buf := NewByteBuf(1)
	buf2 := NewByteBuf(1)
	buf.MustWrite([]byte{1, 2, 3, 4, 5})
	n, err := buf2.ReadFrom(buf)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), n)
	_, v := buf2.ReadAll()
	assert.Equal(t, []byte{1, 2, 3, 4, 5}, v)
}

func TestIOCopy(t *testing.T) {
	buf := NewByteBuf(1)
	buf2 := NewByteBuf(1)
	buf.MustWrite([]byte{1, 2, 3, 4, 5})
	n, err := io.Copy(buf2, buf)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), n)
	_, v := buf2.ReadAll()
	assert.Equal(t, []byte{1, 2, 3, 4, 5}, v)
}

func TestIOCopyWithEOF(t *testing.T) {
	buf := NewByteBuf(1)
	buf2 := NewByteBuf(1)
	n, err := io.Copy(buf2, buf)
	assert.Equal(t, int64(0), n)
	assert.Equal(t, io.EOF, err)
}

func TestGrow(t *testing.T) {
	n := 1024 * 1024
	buf := NewByteBuf(10)
	buf.readerIndex = 1
	buf.writerIndex = 5

	buf.MustWrite(make([]byte, n))
	assert.Equal(t, 0, buf.readerIndex)
	assert.Equal(t, n+4, buf.writerIndex)
}

func TestGrowWithDisableResetReadAndWrite(t *testing.T) {
	n := 1024 * 1024
	buf := NewByteBuf(10, WithDisableResetReadAndWriteIndexAfterGrow(true))
	buf.readerIndex = 1
	buf.writerIndex = 5

	buf.MustWrite(make([]byte, n))
	assert.Equal(t, 1, buf.readerIndex)
	assert.Equal(t, 5+n, buf.GetWriteIndex())
}
