package simple

import (
	"io"

	"github.com/fagongzi/goetty/v2/buf"
	"github.com/fagongzi/goetty/v2/codec"
	"github.com/fagongzi/goetty/v2/codec/length"
)

// NewStringCodec returns a string codec
func NewStringCodec() codec.Codec {
	return length.New(&stringCodec{})
}

type stringCodec struct {
}

func (c *stringCodec) Decode(in *buf.ByteBuf) (interface{}, bool, error) {
	return string(in.ReadMarkedData()), true, nil
}

func (c *stringCodec) Encode(data interface{}, out *buf.ByteBuf, conn io.Writer) error {
	msg, _ := data.(string)
	bytes := []byte(msg)
	out.Write(bytes)
	return nil
}
