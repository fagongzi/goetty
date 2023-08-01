package goetty

import (
	"io"
	"net"
)

type bufferedConn[IN, OUT any] struct {
	net.Conn

	reader io.Reader
}

// newBufferedConn returns a wrapped net.Conn that read from IOSession's in-buffer first
func newBufferedConn[IN, OUT any](conn net.Conn, session IOSession[IN, OUT]) *bufferedConn[IN, OUT] {
	reader := io.MultiReader(session.InBuf(), conn)
	return &bufferedConn[IN, OUT]{
		Conn:   conn,
		reader: reader,
	}
}

func (c *bufferedConn[IN, OUT]) Read(b []byte) (n int, err error) {
	return c.reader.Read(b)
}
