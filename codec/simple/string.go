package simple

import (
	"io"

	"github.com/fagongzi/goetty/v2/buf"
	"github.com/fagongzi/goetty/v2/codec"
	"github.com/fagongzi/goetty/v2/codec/length"
)

// NewStringCodec returns a string codec
func NewStringCodec() codec.Codec[string, string] {
	return length.New[string, string](&stringCodec{})
}

type stringCodec struct {
}

func (c *stringCodec) Decode(in *buf.ByteBuf) (string, bool, error) {
	return string(in.ReadMarkedData()), true, nil
}

func (c *stringCodec) Encode(data string, out *buf.ByteBuf, conn io.Writer) error {
	bytes := []byte(data)
	out.Write(bytes)
	return nil
}
