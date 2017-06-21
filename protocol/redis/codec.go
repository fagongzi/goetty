package redis

import (
	"github.com/fagongzi/goetty"
)

type redisDecoder struct {
}

// NewRedisDecoder returns a redis protocol decoder
func NewRedisDecoder() goetty.Decoder {
	return &redisDecoder{}
}

// Decode decode
func (decoder *redisDecoder) Decode(in *goetty.ByteBuf) (bool, interface{}, error) {
	complete, cmd, err := ReadCommand(in)
	if err != nil {
		return true, nil, err
	}

	if !complete {
		return false, nil, nil
	}

	return true, cmd, nil
}

type redisRespDecoder struct {
}

// NewRedisRespDecoder returns a redis protocol cmd response decoder
func NewRedisRespDecoder() goetty.Decoder {
	return &redisRespDecoder{}
}

// Decode decode
func (decoder *redisRespDecoder) Decode(in *goetty.ByteBuf) (bool, interface{}, error) {
	complete, cmd, err := readCommandResp(in)
	if err != nil {
		return true, nil, err
	}

	if !complete {
		return false, nil, nil
	}

	return true, cmd, nil
}
