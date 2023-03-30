package goetty

import "net"

type WrappedConn struct {
	net.Conn
	s IOSession
}

func (c *WrappedConn) Read(b []byte) (n int, err error) {
	if c.s.OutBuf().Readable() > 0 {
		return c.s.OutBuf().Read(b)
	}
	return c.Conn.Read(b)
}
