package buf

import (
	"fmt"
	"io"

	"github.com/fagongzi/util/hack"
)

const (
	defaultMinGrowSize      = 256
	defaultIOCopyBufferSize = 1024 * 4
)

// Option bytebuf option
type Option func(*ByteBuf)

// WithMemAllocator Set the memory allocator, when Bytebuf is initialized, it needs to
// allocate a []byte of the size specified by capacity from memory. When ByteBuf.Release
// is called, the memory will be freed back to the allocator.
func WithMemAllocator(alloc Allocator) Option {
	return func(bb *ByteBuf) {
		bb.options.alloc = alloc
	}
}

// WithMinGowSize set minimum Grow size. When there is not enough space left
// in the Bytebuf, write data needs to be expanded.
func WithMinGowSize(minGrowSize int) Option {
	return func(bb *ByteBuf) {
		bb.options.minGrowSize = minGrowSize
	}
}

// WithIOCopyBufferSize set io copy buffer used to control how much data will written
// at a time.
func WithIOCopyBufferSize(value int) Option {
	return func(bb *ByteBuf) {
		bb.options.ioCopyBufferSize = value
	}
}

// WithDisableResetReadAndWriteIndexAfterGrow disable reset read and write index after
// grow.
func WithDisableResetReadAndWriteIndexAfterGrow(value bool) Option {
	return func(bb *ByteBuf) {
		bb.options.disableResetReadAndWriteIndexAfterGrow = value
	}
}

// Slice the slice of byte buf
type Slice struct {
	from, to int // [from, to)
	buf      *ByteBuf
}

// Data data
func (s Slice) Data() []byte {
	return s.buf.buf[s.from:s.to]
}

var (
	_ io.WriterTo   = (*ByteBuf)(nil)
	_ io.Writer     = (*ByteBuf)(nil)
	_ io.Reader     = (*ByteBuf)(nil)
	_ io.ReaderFrom = (*ByteBuf)(nil)
)

// ByteBuf is a reusable buffer that holds an internal []byte and maintains 2 indexes for
// read and write data.
//
// | discardable bytes  |   readable bytes   |   writeable bytes  |
// |                    |                    |                    |
// |                    |                    |                    |
// 0      <=       readerIndex    <=     writerIndex    <=     capacity
//
// The ByteBuf implemented io.Reader, io.Writer, io.WriterTo, io.ReaderFrom interface
type ByteBuf struct {
	buf         []byte // buf data, auto +/- size
	readerIndex int
	writerIndex int
	markedIndex int

	options struct {
		alloc                                  Allocator
		minGrowSize                            int
		ioCopyBufferSize                       int
		disableResetReadAndWriteIndexAfterGrow bool
	}
}

// NewByteBuf create bytebuf with options
func NewByteBuf(capacity int, opts ...Option) *ByteBuf {
	b := &ByteBuf{
		readerIndex: 0,
		writerIndex: 0,
	}
	for _, opt := range opts {
		opt(b)
	}
	b.adjust()
	b.buf = b.options.alloc.Alloc(capacity)
	return b
}

func (b *ByteBuf) adjust() {
	if b.options.alloc == nil {
		b.options.alloc = newNonReusableAllocator()
	}
	if b.options.minGrowSize == 0 {
		b.options.minGrowSize = defaultMinGrowSize
	}
	if b.options.ioCopyBufferSize == 0 {
		b.options.ioCopyBufferSize = defaultIOCopyBufferSize
	}
}

// Close close the ByteBuf
func (b *ByteBuf) Close() {
	b.options.alloc.Free(b.buf)
	b.buf = nil
}

// Reset reset to reuse.
func (b *ByteBuf) Reset() {
	b.readerIndex = 0
	b.writerIndex = 0
	b.markedIndex = 0
}

// SetReadIndex set the reader index. The data in the [readIndex, writeIndex] that can be read.
func (b *ByteBuf) SetReadIndex(readIndex int) {
	if readIndex < 0 || readIndex > b.writerIndex {
		panic(fmt.Sprintf("invalid readIndex %d, writeIndex %d", readIndex, b.writerIndex))
	}

	b.readerIndex = readIndex
}

// GetReadIndex returns the read index
func (b *ByteBuf) GetReadIndex() int {
	return b.readerIndex
}

// SetWriteIndex set the write index. The data can write into range [writeIndex, len(buf)).
func (b *ByteBuf) SetWriteIndex(writeIndex int) {
	if writeIndex < b.readerIndex || writeIndex > b.capacity() {
		panic(fmt.Sprintf("invalid writeIndex %d, capacity %d, readIndex %d",
			writeIndex, b.capacity(), b.readerIndex))
	}

	b.writerIndex = writeIndex
}

// GetWriteIndex get the write index
func (b *ByteBuf) GetWriteIndex() int {
	return b.writerIndex
}

// SetMarkIndex mark data in range [readIndex, markIndex)
func (b *ByteBuf) SetMarkIndex(markIndex int) {
	if markIndex > b.writerIndex || markIndex <= b.readerIndex {
		panic(fmt.Sprintf("invalid markIndex %d, readIndex %d, writeIndex %d",
			markIndex, b.readerIndex, b.writerIndex))
	}
	b.markedIndex = markIndex
}

// GetMarkIndex returns the markIndex.
func (b *ByteBuf) GetMarkIndex() int {
	return b.markedIndex
}

// ClearMark clear mark index
func (b *ByteBuf) ClearMark() {
	b.markedIndex = 0
}

// GetMarkedDataLen returns len of marked data
func (b *ByteBuf) GetMarkedDataLen() int {
	return b.markedIndex - b.readerIndex
}

// Skip skip [readIndex, readIndex+n).
func (b *ByteBuf) Skip(n int) {
	if n > b.Readable() {
		panic(fmt.Sprintf("invalid skip %d", n))
	}
	b.readerIndex += n
}

// Slice returns a read only bytebuf slice. ByteBuf may be continuously written to, causing the
// internal buf to reapply, thus invalidating the sliced data in buf[s:e]. Slice only records the
// starting location of the data, and it is safe to read the data when it is certain that the ByteBuf
// will not be written to.
func (b *ByteBuf) Slice(from, to int) Slice {
	if from >= to || to > b.writerIndex {
		panic(fmt.Sprintf("invalid slice by range [%d, %d), writeIndex %d",
			from, to, b.writerIndex))
	}
	return Slice{from, to, b}
}

// RawSlice returns raw buf in range [from, to).  This method requires special care, as the ByteBuf may
// free the internal []byte after the data is written again, causing the slice to fail.
func (b *ByteBuf) RawSlice(from, to int) []byte {
	if from >= to || to > b.writerIndex {
		panic(fmt.Sprintf("invalid slice by range [%d, %d), writeIndex %d",
			from, to, b.writerIndex))
	}
	return b.buf[from:to]
}

// RawBuf returns raw buf. This method requires special care, as the ByteBuf may free the internal []byte
// after the data is written again, causing the slice to fail.
func (b *ByteBuf) RawBuf() []byte {
	return b.buf
}

// Readable return the number of bytes that can be read.
func (b *ByteBuf) Readable() int {
	return b.writerIndex - b.readerIndex
}

// ReadByte read a byte from buf
func (b *ByteBuf) ReadByte() (byte, error) {
	if b.Readable() == 0 {
		return 0, io.EOF
	}

	v := b.buf[b.readerIndex]
	b.readerIndex++
	return v, nil
}

// MustReadByte is similar to ReadByte, buf panic if error retrurned
func (b *ByteBuf) MustReadByte() byte {
	v, err := b.ReadByte()
	if err != nil {
		panic(err)
	}
	return v
}

// ReadBytes read bytes from buf. It's will copy the data to a new byte array.
func (b *ByteBuf) ReadBytes(n int) (readed int, data []byte) {
	readed = n
	if readed > b.Readable() {
		readed = b.Readable()
	}
	if readed == 0 {
		return
	}

	data = make([]byte, readed)
	copy(data, b.buf[b.readerIndex:b.readerIndex+readed])
	b.readerIndex += readed
	return
}

// ReadMarkedData returns [readIndex, markIndex) data
func (b *ByteBuf) ReadMarkedData() []byte {
	_, data := b.ReadBytes(b.GetMarkedDataLen())
	b.ClearMark()
	return data
}

// ReadAll read all readable bytes.
func (b *ByteBuf) ReadAll() (readed int, data []byte) {
	return b.ReadBytes(b.Readable())
}

// ReadInt get int value from buf
func (b *ByteBuf) ReadInt() int {
	if b.Readable() < 4 {
		panic(fmt.Sprintf("read int, but readable is %d", b.Readable()))
	}

	b.readerIndex += 4
	return Byte2Int(b.buf[b.readerIndex-4 : b.readerIndex])
}

// PeekInt is similar to ReadInt, but keep readIndex not changed.
func (b *ByteBuf) PeekInt(offset int) int {
	if b.Readable() < 4 {
		panic(fmt.Sprintf("peek int, but readable is %d", b.Readable()))
	}

	start := b.readerIndex + offset
	return Byte2Int(b.buf[start : start+4])
}

// PeekN is similar to ReadBytes, but keep readIndex not changed.
func (b *ByteBuf) PeekN(offset, bytes int) []byte {
	if b.Readable() < bytes {
		panic(fmt.Sprintf("peek bytes %d, but readable is %d",
			bytes, b.Readable()))
	}

	start := b.readerIndex + offset
	return b.buf[start : start+bytes]
}

// ReadUint16 get uint16 value from buf
func (b *ByteBuf) ReadUint16() uint16 {
	if b.Readable() < 2 {
		panic(fmt.Sprintf("read uint16, but readable is %d", b.Readable()))
	}

	b.readerIndex += 2
	return Byte2Uint16(b.buf[b.readerIndex-2 : b.readerIndex])
}

// ReadUint32 get uint32 value from buf
func (b *ByteBuf) ReadUint32() uint32 {
	if b.Readable() < 4 {
		panic(fmt.Sprintf("read uint32, but readable is %d", b.Readable()))
	}

	b.readerIndex += 4
	return Byte2Uint32(b.buf[b.readerIndex-4 : b.readerIndex])
}

// ReadInt64 get int64 value from buf
func (b *ByteBuf) ReadInt64() int64 {
	if b.Readable() < 8 {
		panic(fmt.Sprintf("read int64, but readable is %d", b.Readable()))
	}

	b.readerIndex += 8
	return Byte2Int64(b.buf[b.readerIndex-8 : b.readerIndex])
}

// ReadUint64 get uint64 value from buf
func (b *ByteBuf) ReadUint64() uint64 {
	if b.Readable() < 8 {
		panic(fmt.Sprintf("read uint64, but readable is %d", b.Readable()))
	}

	b.readerIndex += 8
	return Byte2Uint64(b.buf[b.readerIndex-8 : b.readerIndex])
}

// Writeable return how many bytes can be wirte into buf
func (b *ByteBuf) Writeable() int {
	return b.capacity() - b.writerIndex
}

// MustWrite is similar to Write, but panic if encounter an error.
func (b *ByteBuf) MustWrite(value []byte) {
	if _, err := b.Write(value); err != nil {
		panic(err)
	}
}

// WriteUint16 write uint16 into buf
func (b *ByteBuf) WriteUint16(v uint16) {
	b.Grow(2)
	Uint16ToBytesTo(v, b.buf[b.writerIndex:b.writerIndex+2])
	b.writerIndex += 2
}

// WriteInt write int into buf
func (b *ByteBuf) WriteInt(v int) {
	b.Grow(4)
	Int2BytesTo(v, b.buf[b.writerIndex:b.writerIndex+4])
	b.writerIndex += 4
}

// WriteUint32 write uint32 into buf
func (b *ByteBuf) WriteUint32(v uint32) {
	b.Grow(4)
	Uint32ToBytesTo(v, b.buf[b.writerIndex:b.writerIndex+4])
	b.writerIndex += 4
}

// WriteInt64 write int64 into buf
func (b *ByteBuf) WriteInt64(v int64) {
	b.Grow(8)
	Int64ToBytesTo(v, b.buf[b.writerIndex:b.writerIndex+8])
	b.writerIndex += 8
}

// WriteUint64 write uint64 into buf
func (b *ByteBuf) WriteUint64(v uint64) {
	b.Grow(8)
	Uint64ToBytesTo(v, b.buf[b.writerIndex:b.writerIndex+8])
	b.writerIndex += 8
}

// WriteByte write a byte value into buf.
func (b *ByteBuf) WriteByte(v byte) error {
	b.Grow(1)
	b.buf[b.writerIndex] = v
	b.writerIndex++
	return nil
}

// MustWriteByte is similar to WriteByte, but panic if has any error
func (b *ByteBuf) MustWriteByte(v byte) {
	if err := b.WriteByte(v); err != nil {
		panic(err)
	}
}

// WriteString write a string value to buf
func (b *ByteBuf) WriteString(v string) {
	b.Write(hack.StringToSlice(v))
}

// Grow grow buf size
func (b *ByteBuf) Grow(n int) {
	if free := b.Writeable(); free < n {
		current := b.capacity()
		step := current / 2
		if step < b.options.minGrowSize {
			step = b.options.minGrowSize
		}

		size := current + (n - free)
		target := current
		for {
			if target > size {
				break
			}

			target += step
		}

		newBuf := b.options.alloc.Alloc(target)
		if b.options.disableResetReadAndWriteIndexAfterGrow {
			copy(newBuf, b.buf)
		} else {
			offset := b.writerIndex - b.readerIndex
			copy(newBuf, b.buf[b.readerIndex:b.writerIndex])
			b.readerIndex = 0
			b.writerIndex = offset
		}

		b.options.alloc.Free(b.buf)
		b.buf = newBuf
	}
}

// Write implemented io.Writer interface
func (b *ByteBuf) Write(src []byte) (int, error) {
	n := len(src)
	b.Grow(n)
	copy(b.buf[b.writerIndex:], src)
	b.writerIndex += n
	return n, nil
}

// WriteTo implemented io.WriterTo interface
func (b *ByteBuf) WriteTo(dst io.Writer) (int64, error) {
	n := b.Readable()
	if n == 0 {
		return 0, io.EOF
	}
	if err := WriteTo(b.buf[b.readerIndex:b.writerIndex], dst, b.options.ioCopyBufferSize); err != nil {
		return 0, err
	}
	b.readerIndex = b.writerIndex
	return int64(n), nil
}

// Read implemented io.Reader interface. return n, nil or 0, io.EOF is successful
func (b *ByteBuf) Read(dst []byte) (int, error) {
	if len(dst) == 0 {
		return 0, nil
	}
	n := b.Readable()
	if n == 0 {
		return 0, io.EOF
	}
	if n > len(dst) {
		n = len(dst)
	}
	copy(dst, b.buf[b.readerIndex:b.readerIndex+n])
	b.readerIndex += n
	return n, nil
}

// ReadFrom implemented io.ReaderFrom interface
func (b *ByteBuf) ReadFrom(r io.Reader) (n int64, err error) {
	for {
		b.Grow(b.options.ioCopyBufferSize)
		m, e := r.Read(b.buf[b.writerIndex : b.writerIndex+b.options.ioCopyBufferSize])
		if m < 0 {
			panic("bug: negative Read")
		}

		b.writerIndex += m
		n += int64(m)
		if e == io.EOF {
			return n, nil // e is EOF, so return nil explicitly
		}
		if e != nil {
			return n, e
		}

		if m > 0 {
			return n, e
		}
	}
}

func (b *ByteBuf) capacity() int {
	return len(b.buf)
}

// WriteTo write data to io.Writer, copyBuffer used to control how much data will written
// at a time.
func WriteTo(data []byte, conn io.Writer, copyBuffer int) error {
	if copyBuffer == 0 || copyBuffer > len(data) {
		copyBuffer = len(data)
	}

	written := 0
	total := len(data)
	var err error
	for {
		to := written + copyBuffer
		if to > total {
			to = total
		}

		n, e := conn.Write(data[written:to])
		if n < 0 {
			panic("invalid write")
		}
		written += n
		if e != nil {
			err = e
			break
		}

		if written == total {
			break
		}
	}
	return err
}
