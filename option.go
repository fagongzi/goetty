package goetty

const (
	// DefaultSessionBucketSize default bucket size of session map
	DefaultSessionBucketSize = uint64(64)
	// DefaultReadBuf read buf size
	DefaultReadBuf = 256
	// DefaultWriteBuf write buf size
	DefaultWriteBuf = 256
	// DefaultReadCopyBuf io.CopyBuffer buffer size for read
	DefaultReadCopyBuf = 1024
	// DefaultWriteCopyBuf io.CopyBuffer buffer size for write
	DefaultWriteCopyBuf = 1024
)

// IOSessionAware io session aware
type IOSessionAware interface {
	// Created session created
	Created(IOSession)
	//Closed session closed
	Closed(IOSession)
}
