package goetty

// timer wheel impl
// author:  zhangxu
// date:    2015-6-26
// version: 1.0.0

import (
	"container/list"
	"hash/fnv"
	"math"
	"sync"
	"time"
)

// SimpleTimeWheel Simple time wheel impl
type SimpleTimeWheel struct {
	timer *time.Ticker

	tick        time.Duration
	periodCount int64 // 每轮多少次
	pos         int64 // 当前指针
	period      int64 // 轮数

	callbacks  map[string]func(key string)
	timeoutMap map[int64]*list.List

	mutex *sync.Mutex
}

// NewSimpleTimeWheel create a simple time wheel
func NewSimpleTimeWheel(tick time.Duration, periodCount int64) *SimpleTimeWheel {
	timeWheel := &SimpleTimeWheel{
		timer:       time.NewTicker(tick),
		tick:        tick,
		periodCount: periodCount,
		pos:         0,
		period:      0,

		callbacks:  make(map[string]func(key string)),
		timeoutMap: make(map[int64]*list.List),

		mutex: &sync.Mutex{},
	}

	return timeWheel
}

// AddWithID add a timeout call with ID
func (t *SimpleTimeWheel) AddWithID(timeout time.Duration, key string, callback func(key string)) {
	t.mutex.Lock()

	index := t.passed() + int64(float64(timeout.Nanoseconds())/float64(t.tick.Nanoseconds())+0.5)

	l, ok := t.timeoutMap[index]
	if !ok {
		l = list.New()
		t.timeoutMap[index] = l
	}

	l.PushBack(key)
	t.callbacks[key] = callback
	t.mutex.Unlock()
}

// Add add a timeout callback
func (t *SimpleTimeWheel) Add(timeout time.Duration, callback func(key string)) string {
	key := NewKey()
	t.AddWithID(timeout, key, callback)
	return key
}

// Cancel cancel a callback
func (t *SimpleTimeWheel) Cancel(key string) {
	t.mutex.Lock()
	delete(t.callbacks, key)
	t.mutex.Unlock()
}

// Start start a timeout calc
func (t *SimpleTimeWheel) Start() {
	go func() {
		for {
			<-t.timer.C
			t.turn()
		}
	}()
}

// Stop stop
func (t *SimpleTimeWheel) Stop() {
	if nil != t.timer {
		t.timer.Stop()
	}
}

func (t *SimpleTimeWheel) turn() {
	t.mutex.Lock()
	t.mutex.Unlock()

	t.pos++

	if t.pos == t.periodCount {
		t.pos = 0
		t.period++
	}

	t.doTimeout()
}

func (t *SimpleTimeWheel) doTimeout() {
	timeKey := t.passed()

	keys, ok := t.timeoutMap[timeKey]

	if ok {
		for iter := keys.Front(); iter != nil; iter = iter.Next() {
			key, _ := iter.Value.(string)
			f, _ := t.callbacks[key]
			if nil != f {
				delete(t.callbacks, key)
				f(key)
			}
		}
	}

	delete(t.timeoutMap, timeKey)
}

func (t *SimpleTimeWheel) passed() int64 {
	return t.periodCount*t.period + t.pos
}

// HashedTimeWheel hash Simple time wheel
type HashedTimeWheel struct {
	mask        int
	wheelBucket []*SimpleTimeWheel
}

// NewHashedTimeWheel create a new HashedTimeWheel
func NewHashedTimeWheel(duration time.Duration, periodCount int64, powOf2 int) *HashedTimeWheel {
	max := int(math.Pow(2.0, float64(powOf2)))
	h := &HashedTimeWheel{
		mask:        max - 1,
		wheelBucket: make([]*SimpleTimeWheel, max),
	}

	h.init(duration, periodCount, max)

	return h
}

// Add add a timeout calc
func (h *HashedTimeWheel) Add(timeout time.Duration, callback func(key string)) string {
	key := NewKey()
	h.AddWithID(timeout, key, callback)
	return key
}

// AddWithID add a timeout calc by ID
func (h *HashedTimeWheel) AddWithID(timeout time.Duration, key string, callback func(key string)) string {
	index := hashCode(key) & h.mask
	h.wheelBucket[index].AddWithID(timeout, key, callback)
	return key
}

// Cancel calc a timeout
func (h *HashedTimeWheel) Cancel(key string) {
	index := hashCode(key) & h.mask
	h.wheelBucket[index].Cancel(key)
}

func (h *HashedTimeWheel) init(duration time.Duration, periodCount int64, max int) {
	for i := 0; i < max; i++ {
		h.wheelBucket[i] = NewSimpleTimeWheel(duration, periodCount)
	}
}

// Start start timeout calc
func (h *HashedTimeWheel) Start() {
	for i := 0; i < len(h.wheelBucket); i++ {
		go h.wheelBucket[i].Start()
	}
}

// Stop stop timeout calc
func (h *HashedTimeWheel) Stop() {
	for i := 0; i < len(h.wheelBucket); i++ {
		go h.wheelBucket[i].Stop()
	}
}

// HashCode get has code
func HashCode(v string) int {
	return hashCode(v)
}

func hashCode(v string) int {
	h := fnv.New32a()
	h.Write([]byte(v))
	code := h.Sum32()
	return int(code)
}
