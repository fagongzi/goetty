package buf

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSyncPoolAllocSmall(t *testing.T) {
	pool := NewSyncPool(128, 1024, 2)
	mem := pool.Alloc(64)
	assert.Equal(t, 64, len(mem))
	assert.Equal(t, 128, cap(mem))
	pool.Free(mem)
}

func TestSyncPoolAllocLarge(t *testing.T) {
	pool := NewSyncPool(128, 1024, 2)
	mem := pool.Alloc(2048)
	assert.Equal(t, 2048, len(mem))
	assert.Equal(t, 2048, cap(mem))
	pool.Free(mem)
}

func BenchmarkSyncPoolAllocAndFree128(b *testing.B) {
	pool := NewSyncPool(128, 1024, 2)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			pool.Free(pool.Alloc(128))
		}
	})
}

func BenchmarkSyncPoolAllocAndFree256(b *testing.B) {
	pool := NewSyncPool(128, 1024, 2)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			pool.Free(pool.Alloc(256))
		}
	})
}

func BenchmarkSyncPoolAllocAndFree512(b *testing.B) {
	pool := NewSyncPool(128, 1024, 2)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			pool.Free(pool.Alloc(512))
		}
	})
}

func BenchmarkSyncPoolCacheMiss128(b *testing.B) {
	pool := NewSyncPool(128, 1024, 2)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			pool.Alloc(128)
		}
	})
}

func Benchmark_SyncPool_CacheMiss_256(b *testing.B) {
	pool := NewSyncPool(128, 1024, 2)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			pool.Alloc(256)
		}
	})
}

func Benchmark_SyncPool_CacheMiss_512(b *testing.B) {
	pool := NewSyncPool(128, 1024, 2)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			pool.Alloc(512)
		}
	})
}

func Benchmark_Make_128(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = make([]byte, 128)
		}
	})
}

func Benchmark_Make_256(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = make([]byte, 256)
		}
	})
}

func Benchmark_Make_512(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = make([]byte, 512)
		}
	})
}
