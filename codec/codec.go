package codec

import (
	"io"

	"github.com/fagongzi/goetty/v3/buf"
)

// Codec message codec, used to encode message to bytes or decode bytes to message.
type Codec[IN any, OUT any] interface {
	// Encode encode message into the out buffer or write directly to the underlying connection.
	Encode(message OUT, out *buf.ByteBuf, conn io.Writer) error
	// Decode decode message from the bytes buffer, returns false if there is not enough data.
	Decode(in *buf.ByteBuf) (message IN, complete bool, err error)
}
