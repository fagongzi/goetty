package buf

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAlloc(t *testing.T) {
	alloc := newNonReusableAllocator()
	assert.Equal(t, 10, len(alloc.Alloc(10)))
}
