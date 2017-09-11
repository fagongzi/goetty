package goetty

const (
	// KB kb
	KB = 1024
	// MB mb
	MB = 1024 * 1024
)

var (
	defaultPool = NewAtomPool(
		256,  // 256 byte
		8*MB, // 1MB
		2,
		64*MB, // 64MB per page
	)
)
