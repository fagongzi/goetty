package buf

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestByte2Int(t *testing.T) {
	assert.Equal(t, 1, Byte2Int(Int2Bytes(1)))
	assert.Equal(t, math.MaxInt32, Byte2Int(Int2Bytes(math.MaxInt32)))
}

func TestByte2Int64(t *testing.T) {
	assert.Equal(t, int64(1), Byte2Int64(Int64ToBytes(1)))
	assert.Equal(t, int64(math.MaxInt64), Byte2Int64(Int64ToBytes(math.MaxInt64)))
}

func TestByte2Uint64(t *testing.T) {
	assert.Equal(t, uint64(1), Byte2Uint64(Uint64ToBytes(1)))
	assert.Equal(t, uint64(math.MaxUint64), Byte2Uint64(Uint64ToBytes(math.MaxUint64)))
}

func TestByte2Uint32(t *testing.T) {
	assert.Equal(t, uint32(1), Byte2Uint32(Uint32ToBytes(1)))
	assert.Equal(t, uint32(math.MaxUint32), Byte2Uint32(Uint32ToBytes(math.MaxUint32)))
}

func TestByte2Uint16(t *testing.T) {
	assert.Equal(t, uint16(1), Byte2Uint16(Uint16ToBytes(1)))
	assert.Equal(t, uint16(math.MaxUint16), Byte2Uint16(Uint16ToBytes(math.MaxUint16)))
}
