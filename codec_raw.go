package goetty

type RawDecoder struct {
}

func (decoder *RawDecoder) Decode(in *ByteBuf) (bool, interface{}, error) {
	_, data, err := in.ReadMarkedBytes()

	if err != nil {
		return true, nil, err
	}

	return true, data, nil
}
