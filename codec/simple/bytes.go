package simple

import (
	"github.com/fagongzi/goetty"
)

// NewBytesCodec returns a bytes codec
func NewBytesCodec() (goetty.Decoder, goetty.Encoder) {
	c := &bytesCodec{}
	return goetty.NewIntLengthFieldBasedDecoder(c),
		goetty.NewIntLengthFieldBasedEncoder(c)
}

type bytesCodec struct {
}

func (c bytesCodec) Decode(in *goetty.ByteBuf) (bool, interface{}, error) {
	_, data, err := in.ReadMarkedBytes()
	if err != nil {
		return false, nil, err
	}

	return true, data, nil
}

func (c bytesCodec) Encode(data interface{}, out *goetty.ByteBuf) error {
	bytes, _ := data.([]byte)
	out.Write(bytes)
	return nil
}
