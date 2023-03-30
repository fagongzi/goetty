package goetty

import (
	"io"
	"net"
)

type WrappedConn struct {
	net.Conn

	reader io.Reader
}

func NewWrappedConn(conn net.Conn, session IOSession) *WrappedConn {
	reader := io.MultiReader(session.InBuf(), conn)
	return &WrappedConn{
		Conn:   conn,
		reader: reader,
	}
}

func (c *WrappedConn) Read(b []byte) (n int, err error) {
	return c.reader.Read(b)
}
