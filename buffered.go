package goetty

import (
	"io"
	"net"
)

type bufferedConn struct {
	net.Conn

	reader io.Reader
}

// newBufferedConn returns a wrapped net.Conn that read from IOSession's in-buffer first
func newBufferedConn(conn net.Conn, session IOSession) *bufferedConn {
	reader := io.MultiReader(session.(BufferedIOSession).InBuf(), conn)
	return &bufferedConn{
		Conn:   conn,
		reader: reader,
	}
}

func (c *bufferedConn) Read(b []byte) (n int, err error) {
	return c.reader.Read(b)
}
