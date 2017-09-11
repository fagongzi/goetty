package goetty

const (
	// KB kb
	KB = 1024
	// MB mb
	MB = 1024 * 1024
)

var (
	mp          Pool
	defaultMin  = 256
	defaultMax  = 8 * MB
	defaultPage = 64 * MB
)

// UseDefaultMemPool use the default mem pool
func UseDefaultMemPool() {
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
