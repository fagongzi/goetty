package length

import (
	"fmt"
	"io"

	"github.com/fagongzi/goetty/v2/buf"
	"github.com/fagongzi/goetty/v2/codec"
)

const (
	fieldLength        = 4
	defaultMaxBodySize = 1024 * 1024 * 10
)

type lengthCodec struct {
	baseCodec           codec.Codec
	lengthFieldOffset   int
	lengthAdjustment    int
	initialBytesToStrip int
	maxBodySize         int
}

// New returns a default IntLengthFieldBased codec
func New(baseCodec codec.Codec) codec.Codec {
	return NewWithSize(baseCodec, 0, 0, 0, defaultMaxBodySize)
}

// NewWithSize  create IntLengthFieldBased codec
// initialBytesToStrip + lengthFieldOffset + 4(length)
// lengthAdjustment, some case as below:
// 1. 0 :                                             base decoder received: body
// 2. -4:                                             base decoder received: 4(length) + body
// 3. -(4 + lengthFieldOffset):                       base decoder received: lengthFieldOffset + 4(length) + body
// 4. -(4 + lengthFieldOffset + initialBytesToStrip): base decoder received: initialBytesToStrip + lengthFieldOffset + 4(length)
func NewWithSize(baseCodec codec.Codec, lengthFieldOffset, lengthAdjustment, initialBytesToStrip, maxBodySize int) codec.Codec {
	return &lengthCodec{
		baseCodec:           baseCodec,
		lengthFieldOffset:   lengthFieldOffset,
		lengthAdjustment:    lengthAdjustment,
		initialBytesToStrip: initialBytesToStrip,
		maxBodySize:         maxBodySize,
	}
}

func (c *lengthCodec) Decode(in *buf.ByteBuf) (any, bool, error) {
	readable := in.Readable()

	minFrameLength := c.initialBytesToStrip + c.lengthFieldOffset + fieldLength
	if readable < minFrameLength {
		return nil, false, nil
	}

	length := in.PeekInt(c.initialBytesToStrip + c.lengthFieldOffset)
	if length > c.maxBodySize {
		return nil, false, fmt.Errorf("too big body size %d, max is %d", length, c.maxBodySize)
	}

	skip := minFrameLength + c.lengthAdjustment
	minFrameLength += length
	if readable < minFrameLength {
		return nil, false, nil
	}

	in.Skip(skip)
	in.SetMarkIndex(length + in.GetReadIndex())
	return c.baseCodec.Decode(in)
}

func (c *lengthCodec) Encode(message any, out *buf.ByteBuf, conn io.Writer) error {
	oldIndex := out.GetWriteIndex()
	out.Grow(4)
	out.SetWriteIndex(oldIndex + 4)
	err := c.baseCodec.Encode(message, out, conn)
	if err != nil {
		return err
	}
	newIndex := out.GetWriteIndex()
	out.SetWriteIndex(oldIndex)
	out.WriteInt(newIndex - oldIndex - 4)
	out.SetWriteIndex(newIndex)
	return nil
}
