package simple

import (
	"io"

	"github.com/fagongzi/goetty/v3/buf"
	"github.com/fagongzi/goetty/v3/codec"
	"github.com/fagongzi/goetty/v3/codec/length"
)

// NewBytesCodec returns a codec to used to encode and decode []byte. It used lengthCodec to add a length
// field to head.
func NewBytesCodec() codec.Codec[[]byte, []byte] {
	return length.New[[]byte, []byte](&bytesCodec{})
}

type bytesCodec struct {
}

func (c *bytesCodec) Decode(in *buf.ByteBuf) ([]byte, bool, error) {
	return in.ReadMarkedData(), true, nil
}

func (c *bytesCodec) Encode(data []byte, out *buf.ByteBuf, conn io.Writer) error {
	out.Write(data)
	return nil
}
