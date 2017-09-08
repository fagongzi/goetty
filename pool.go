package goetty

var (
	defaultPool = NewAtomPool(
		128,     //128 byte
		64*1024, // 64kb
		2,
		1024*1024, // 1MB per page
	)
)
