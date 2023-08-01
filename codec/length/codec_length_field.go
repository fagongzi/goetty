package length

import (
	"fmt"
	"io"

	"github.com/fagongzi/goetty/v3/buf"
	"github.com/fagongzi/goetty/v3/codec"
)

const (
	fieldLength        = 4
	defaultMaxBodySize = 1024 * 1024 * 10
)

type lengthCodec[IN any, OUT any] struct {
	baseCodec           codec.Codec[IN, OUT]
	lengthFieldOffset   int
	lengthAdjustment    int
	initialBytesToStrip int
	maxBodySize         int
}

// New returns a default IntLengthFieldBased codec
func New[IN any, OUT any](baseCodec codec.Codec[IN, OUT]) codec.Codec[IN, OUT] {
	return NewWithSize(baseCodec, 0, 0, 0, defaultMaxBodySize)
}

// NewWithSize  create IntLengthFieldBased codec
// initialBytesToStrip + lengthFieldOffset + 4(length)
// lengthAdjustment, some case as below:
// 1. 0 :                                             base decoder received: body
// 2. -4:                                             base decoder received: 4(length) + body
// 3. -(4 + lengthFieldOffset):                       base decoder received: lengthFieldOffset + 4(length) + body
// 4. -(4 + lengthFieldOffset + initialBytesToStrip): base decoder received: initialBytesToStrip + lengthFieldOffset + 4(length)
func NewWithSize[IN any, OUT any](baseCodec codec.Codec[IN, OUT],
	lengthFieldOffset,
	lengthAdjustment,
	initialBytesToStrip,
	maxBodySize int) codec.Codec[IN, OUT] {
	return &lengthCodec[IN, OUT]{
		baseCodec:           baseCodec,
		lengthFieldOffset:   lengthFieldOffset,
		lengthAdjustment:    lengthAdjustment,
		initialBytesToStrip: initialBytesToStrip,
		maxBodySize:         maxBodySize,
	}
}

func (c *lengthCodec[IN, OUT]) Decode(in *buf.ByteBuf) (IN, bool, error) {
	var msg IN
	readable := in.Readable()

	minFrameLength := c.initialBytesToStrip + c.lengthFieldOffset + fieldLength
	if readable < minFrameLength {
		return msg, false, nil
	}

	length := in.PeekInt(c.initialBytesToStrip + c.lengthFieldOffset)
	if length > c.maxBodySize {
		return msg, false, fmt.Errorf("too big body size %d, max is %d", length, c.maxBodySize)
	}
	if length <= 0 {
		return msg, false, fmt.Errorf("invalid body size: %d", length)
	}

	skip := minFrameLength + c.lengthAdjustment
	minFrameLength += length
	if readable < minFrameLength {
		return msg, false, nil
	}

	in.Skip(skip)
	in.SetMarkIndex(length + in.GetReadIndex())
	return c.baseCodec.Decode(in)
}

func (c *lengthCodec[IN, OUT]) Encode(message OUT, out *buf.ByteBuf, conn io.Writer) error {
	oldIndexOffset := out.Readable()
	out.Grow(4)
	out.SetWriteIndex(out.GetReadIndex() + oldIndexOffset + 4)
	err := c.baseCodec.Encode(message, out, conn)
	if err != nil {
		return err
	}
	newIndex := out.GetWriteIndex()
	oldIndex := out.GetReadIndex() + oldIndexOffset
	out.SetWriteIndex(oldIndex)
	out.WriteInt(newIndex - oldIndex - 4)
	out.SetWriteIndex(newIndex)
	return nil
}
