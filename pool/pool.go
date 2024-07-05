package pool

import (
	"sort"
	"sync"
	"sync/atomic"
)

const (
	minBitSize = 6 // 2**6=64 is a CPU cache line size
	steps      = 20

	minSize = 1 << minBitSize
	maxSize = 1 << (minBitSize + steps - 1)

	calibrateCallsThreshold = 42000
	maxPercentile           = 0.95
)

type Pool[T any] interface {
	Get() T
	Put(T)
	Count() int64
}

type pool[T any] struct {
	internalPool sync.Pool
	count        int64
	calibrating  uint64
	calls        [steps]uint64
	defaultSize  uint64
	maxSize      uint64
	pool         sync.Pool
}

type callSize struct {
	calls uint64
	size  uint64
}

type callSizes []callSize

func getZero[T any]() T {
	var r T
	return r
}

var defaultPool = &pool[any]{}

func Get() any { return defaultPool.Get() }

func Put(b any) { defaultPool.Put(b) }

func (p *pool[T]) Get() T {
	val := p.internalPool.Get()
	if val == nil {
		return getZero[T]()
	}

	atomic.AddInt64(&p.count, -1)

	return val.(T)
}

func (p *pool[T]) Put(value T) {
	p.internalPool.Put(value)
	atomic.AddInt64(&p.count, 1)
}

func (p *pool[T]) Count() int64 {
	return atomic.LoadInt64(&p.count)
}

func NewPool[T any]() Pool[T] {
	return &pool[T]{
		internalPool: sync.Pool{},
	}
}

func (p *pool[T]) calibrate() {
	if !atomic.CompareAndSwapUint64(&p.calibrating, 0, 1) {
		return
	}

	a := make(callSizes, 0, steps)
	var callsSum uint64
	for i := uint64(0); i < steps; i++ {
		calls := atomic.SwapUint64(&p.calls[i], 0)
		callsSum += calls
		a = append(a, callSize{
			calls: calls,
			size:  minSize << i,
		})
	}
	sort.Sort(a)

	defaultSize := a[0].size
	maxSize := defaultSize

	maxSum := uint64(float64(callsSum) * maxPercentile)
	callsSum = 0
	for i := 0; i < steps; i++ {
		if callsSum > maxSum {
			break
		}
		callsSum += a[i].calls
		size := a[i].size
		if size > maxSize {
			maxSize = size
		}
	}

	atomic.StoreUint64(&p.defaultSize, defaultSize)
	atomic.StoreUint64(&p.maxSize, maxSize)

	atomic.StoreUint64(&p.calibrating, 0)
}
func (ci callSizes) Len() int {
	return len(ci)
}

func (ci callSizes) Swap(i, j int) {
	ci[i], ci[j] = ci[j], ci[i]
}

func (ci callSizes) Less(i, j int) bool {
	return ci[i].calls > ci[j].calls
}

func index(n int) int {
	n--
	n >>= minBitSize
	idx := 0
	for n > 0 {
		n >>= 1
		idx++
	}
	if idx >= steps {
		idx = steps - 1
	}
	return idx
}
