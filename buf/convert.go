package buf

import (
	"encoding/binary"
)

// Byte2Int byte array to int value using big order
func Byte2Int(data []byte) int {
	return (int(data[0])&0xff)<<24 |
		(int(data[1])&0xff)<<16 |
		(int(data[2])&0xff)<<8 |
		(int(data[3]) & 0xff)
}

// Byte2Int64 byte array to int64 value using big order
func Byte2Int64(data []byte) int64 {
	return (int64(data[0])&0xff)<<56 |
		(int64(data[1])&0xff)<<48 |
		(int64(data[2])&0xff)<<40 |
		(int64(data[3])&0xff)<<32 |
		(int64(data[4])&0xff)<<24 |
		(int64(data[5])&0xff)<<16 |
		(int64(data[6])&0xff)<<8 |
		(int64(data[7]) & 0xff)
}

// Byte2Uint64 byte array to int64 value using big order
func Byte2Uint64(data []byte) uint64 {
	return binary.BigEndian.Uint64(data)
}

// Byte2Uint16 byte array to uint16 value using big order
func Byte2Uint16(data []byte) uint16 {
	return binary.BigEndian.Uint16(data)
}

// Byte2Uint32 byte array to uint32 value using big order
func Byte2Uint32(data []byte) uint32 {
	return binary.BigEndian.Uint32(data)
}

// Int2BytesTo int value to bytes array using big order
func Int2BytesTo(v int, ret []byte) {
	ret[0] = byte(v >> 24)
	ret[1] = byte(v >> 16)
	ret[2] = byte(v >> 8)
	ret[3] = byte(v)
}

// Int2Bytes int value to bytes array using big order
func Int2Bytes(v int) []byte {
	ret := make([]byte, 4)
	Int2BytesTo(v, ret)
	return ret
}

// Int64ToBytesTo int64 value to bytes array using big order
func Int64ToBytesTo(v int64, ret []byte) {
	ret[0] = byte(v >> 56)
	ret[1] = byte(v >> 48)
	ret[2] = byte(v >> 40)
	ret[3] = byte(v >> 32)
	ret[4] = byte(v >> 24)
	ret[5] = byte(v >> 16)
	ret[6] = byte(v >> 8)
	ret[7] = byte(v)
}

// Uint64ToBytesTo uint64 value to bytes array using big order
func Uint64ToBytesTo(v uint64, ret []byte) {
	binary.BigEndian.PutUint64(ret, v)
}

// Int64ToBytes int64 value to bytes array using big order
func Int64ToBytes(v int64) []byte {
	ret := make([]byte, 8)
	Int64ToBytesTo(v, ret)
	return ret
}

// Uint64ToBytes uint64 value to bytes array using big order
func Uint64ToBytes(v uint64) []byte {
	ret := make([]byte, 8)
	Uint64ToBytesTo(v, ret)
	return ret
}

// Uint32ToBytesTo uint32 value to bytes array using big order
func Uint32ToBytesTo(v uint32, ret []byte) {
	binary.BigEndian.PutUint32(ret, v)
}

// Uint32ToBytes uint32 value to bytes array using big order
func Uint32ToBytes(v uint32) []byte {
	ret := make([]byte, 4)
	Uint32ToBytesTo(v, ret)
	return ret
}

// Uint16ToBytesTo uint16 value to bytes array using big order
func Uint16ToBytesTo(v uint16, ret []byte) {
	binary.BigEndian.PutUint16(ret, v)
}

// Uint16ToBytes uint16 value to bytes array using big order
func Uint16ToBytes(v uint16) []byte {
	ret := make([]byte, 2)
	Uint16ToBytesTo(v, ret)
	return ret
}
