package buf

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAllocate(t *testing.T) {
	allocator := newNonReusableAllocator()
	assert.Equal(t, 10, len(allocator.Allocate(10)))
}
