package simple

import (
	"github.com/fagongzi/goetty/buf"
	"github.com/fagongzi/goetty/codec"
	"github.com/fagongzi/goetty/codec/length"
)

// NewStringCodec returns a string codec
func NewStringCodec() (codec.Encoder, codec.Decoder) {
	c := &stringCodec{}
	return length.New(c, c)
}

type stringCodec struct {
}

func (c stringCodec) Decode(in *buf.ByteBuf) (bool, interface{}, error) {
	_, data, err := in.ReadMarkedBytes()

	if err != nil {
		return false, "", err
	}

	return true, string(data), nil
}

func (c stringCodec) Encode(data interface{}, out *buf.ByteBuf) error {
	msg, _ := data.(string)
	bytes := []byte(msg)
	out.Write(bytes)
	return nil
}
