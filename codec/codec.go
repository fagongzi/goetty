package codec

import (
	"io"

	"github.com/fagongzi/goetty/v2/buf"
)

// Codec message codec, used to encode message to bytes or decode bytes to message.
type Codec interface {
	// Encode encode message into the out buffer or write directly to the underlying connection.
	Encode(message any, out *buf.ByteBuf, conn io.Writer) error
	// Decode decode message from the bytes buffer, returns false if there is not enough data.
	Decode(in *buf.ByteBuf) (message any, complete bool, err error)
}
