package redis

import (
	"github.com/fagongzi/goetty/v2/buf"
	"github.com/fagongzi/goetty/v2/codec"
)

type redisDecoder struct {
}

// NewRedisDecoder returns a redis protocol decoder
func NewRedisDecoder() codec.Decoder {
	return &redisDecoder{}
}

// Decode decode
func (decoder *redisDecoder) Decode(in *buf.ByteBuf) (bool, interface{}, error) {
	complete, cmd, err := ReadCommand(in)
	if err != nil {
		return true, nil, err
	}

	if !complete {
		return false, nil, nil
	}

	return true, cmd, nil
}

type redisReplyDecoder struct {
}

// NewRedisReplyDecoder returns a redis protocol cmd reply decoder
func NewRedisReplyDecoder() codec.Decoder {
	return &redisReplyDecoder{}
}

// Decode decode
func (decoder *redisReplyDecoder) Decode(in *buf.ByteBuf) (bool, interface{}, error) {
	complete, cmd, err := readCommandReply(in)
	if err != nil {
		return true, nil, err
	}

	if !complete {
		return false, nil, nil
	}

	return true, cmd, nil
}
