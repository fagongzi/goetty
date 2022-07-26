package simple

import (
	"io"

	"github.com/fagongzi/goetty/v2/buf"
	"github.com/fagongzi/goetty/v2/codec"
	"github.com/fagongzi/goetty/v2/codec/length"
)

// NewBytesCodec returns a codec to used to encode and decode []byte. It used lengthCodec to add a length
// field to head.
func NewBytesCodec() codec.Codec {
	return length.New(&bytesCodec{})
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
