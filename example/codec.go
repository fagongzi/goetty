package example

import (
	"github.com/fagongzi/goetty"
)

type StringDecoder struct {
}

func (decoder StringDecoder) Decode(in *ByteBuf) (bool, interface{}, error) {
	_, data, err := in.ReadMarkedBytes()

	if err != nil {
		return true, "", err
	}

	return true, string(data), nil
}

type StringEncoder struct {
}

func (self StringEncoder) Encode(data interface{}, out *ByteBuf) error {
	msg, _ := data.(string)
	bytes := []byte(msg)
	out.WriteInt(len(bytes))
	out.Write(bytes)
	return nil
}
