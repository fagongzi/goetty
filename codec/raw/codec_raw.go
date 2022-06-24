package raw

import (
	"github.com/fagongzi/goetty/v2/buf"
	"github.com/fagongzi/goetty/v2/codec"
)

type rawCodec struct {
}

func (c *rawCodec) Decode(in *buf.ByteBuf) (bool, interface{}, error) {
	_, data, err := in.ReadMarkedBytes()

	if err != nil {
		return true, data, err
	}

	return true, data, nil
}

func (c *rawCodec) Encode(data interface{}, out *buf.ByteBuf) error {
	out.Write(data.([]byte))
	return nil
}

// New returns raw codec
func New() (codec.Encoder, codec.Decoder) {
	c := &rawCodec{}
	return c, c
}
