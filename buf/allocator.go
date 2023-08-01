package buf

// Allocator memory allocation for ByteBuf
type Allocator interface {
	// Allocate allocate a []byte with len(data) >= size, and the returned []byte cannot
	// be expanded in use.
	Allocate(capacity int) []byte
	// Free free the allocated memory
	Free([]byte)
}

type nonReusableAllocator struct {
}

func newNonReusableAllocator() Allocator {
	return &nonReusableAllocator{}
}

func (ma *nonReusableAllocator) Allocate(size int) []byte {
	return make([]byte, size)
}

func (ma *nonReusableAllocator) Free([]byte) {

}
