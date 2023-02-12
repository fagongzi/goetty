package goetty

const (
	// defaultSessionBucketSize default bucket size of session map
	defaultSessionBucketSize = uint64(64)
	// defaultReadBuf read buf size
	defaultReadBuf = 256
	// defaultWriteBuf write buf size
	defaultWriteBuf = 256
	// defaultReadCopyBuf io.CopyBuffer buffer size for read
	defaultReadCopyBuf = 1024
	// defaultWriteCopyBuf io.CopyBuffer buffer size for write
	defaultWriteCopyBuf = 1024
)

// IOSessionAware io session aware
type IOSessionAware[IN any, OUT any] interface {
	// Created session created
	Created(IOSession[IN, OUT])
	//Closed session closed
	Closed(IOSession[IN, OUT])
}
