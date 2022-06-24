package simple

import (
	"github.com/fagongzi/goetty/v2/buf"
	"github.com/fagongzi/goetty/v2/codec"
	"github.com/fagongzi/goetty/v2/codec/length"
)

// NewStringCodec returns a string codec
func NewStringCodec() (codec.Encoder, codec.Decoder) {
	c := &stringCodec{}
	return length.New(c, c)
}

type stringCodec struct {
}

func (c stringCodec) Decode(in *buf.ByteBuf) (bool, interface{}, error) {
	v := string(in.GetMarkedRemindData())
	in.MarkedBytesReaded()
	return true, v, nil
}

func (c stringCodec) Encode(data interface{}, out *buf.ByteBuf) error {
	msg, _ := data.(string)
	bytes := []byte(msg)
	out.Write(bytes)
	return nil
}
