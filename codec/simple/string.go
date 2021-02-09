package simple

import (
	"github.com/fagongzi/goetty"
)

// NewStringCodec returns a string codec
func NewStringCodec() (goetty.Decoder, goetty.Encoder) {
	c := &stringCodec{}
	return goetty.NewIntLengthFieldBasedDecoder(c),
		goetty.NewIntLengthFieldBasedEncoder(c)
}

type stringCodec struct {
}

func (c stringCodec) Decode(in *goetty.ByteBuf) (bool, interface{}, error) {
	_, data, err := in.ReadMarkedBytes()

	if err != nil {
		return false, "", err
	}

	return true, string(data), nil
}

func (c stringCodec) Encode(data interface{}, out *goetty.ByteBuf) error {
	msg, _ := data.(string)
	bytes := []byte(msg)
	out.Write(bytes)
	return nil
}
