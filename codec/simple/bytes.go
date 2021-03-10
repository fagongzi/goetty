package simple

import (
	"github.com/fagongzi/goetty/buf"
	"github.com/fagongzi/goetty/codec"
	"github.com/fagongzi/goetty/codec/length"
)

// NewBytesCodec returns a bytes codec
func NewBytesCodec() (codec.Encoder, codec.Decoder) {
	c := &bytesCodec{}
	return length.New(c, c)
}

type bytesCodec struct {
}

func (c bytesCodec) Decode(in *buf.ByteBuf) (bool, interface{}, error) {
	_, data, err := in.ReadMarkedBytes()
	if err != nil {
		return false, nil, err
	}

	return true, data, nil
}

func (c bytesCodec) Encode(data interface{}, out *buf.ByteBuf) error {
	bytes, _ := data.([]byte)
	out.Write(bytes)
	return nil
}
