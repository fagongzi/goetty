package goetty

import (
	"syscall"

	"golang.org/x/sys/unix"
)

func listenControl(network string, address string, conn syscall.RawConn) (err error) {
	return conn.Control(func(fd uintptr) {
		syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, unix.SO_REUSEPORT, 1)
		syscall.SetsockoptInt(int(fd), syscall.SOL_TCP, unix.TCP_FASTOPEN, 1)
	})
}
