package goetty

import (
	"sync"
)

const (
	// KB kb
	KB = 1024
	// MB mb
	MB = 1024 * 1024
)

var (
	lock sync.Mutex
	mp          Pool
	defaultMin  = 256
	defaultMax  = 8 * MB
	defaultPage = 64 * MB
)

func getDefaultMP() Pool {
	lock.Lock()
	if mp == nil {
		useDefaultMemPool()
	}
	lock.Unlock()

	return mp
}

func useDefaultMemPool() {
	mp = NewAtomPool(
		defaultMin,
		defaultMax,
		2,
		defaultPage,
	)
}

// UseMemPool use the custom mem pool
func UseMemPool(min, max, page int) {
	mp = NewAtomPool(
		min,
		max,
		2,
		page,
	)
}
