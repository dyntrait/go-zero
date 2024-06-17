package collection

import (
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/mathx"
	"github.com/zeromicro/go-zero/core/timex"
)

type (
	// BucketInterface is the interface that defines the buckets.
	BucketInterface[T Numerical] interface {
		Add(v T)
		Reset()
	}

	// Numerical is the interface that restricts the numerical type.
	Numerical = mathx.Numerical

	// RollingWindowOption let callers customize the RollingWindow.
	RollingWindowOption[T Numerical, B BucketInterface[T]] func(rollingWindow *RollingWindow[T, B])

	// RollingWindow defines a rolling window to calculate the events in buckets with the time interval.
	RollingWindow[T Numerical, B BucketInterface[T]] struct {
		lock          sync.RWMutex
		size          int
		win           *window[T, B]
		interval      time.Duration
		offset        int
		ignoreCurrent bool
		lastTime      time.Duration // start time of the last bucket
	}
)

// NewRollingWindow returns a RollingWindow that with size buckets and time interval,
// use opts to customize the RollingWindow.
func NewRollingWindow[T Numerical, B BucketInterface[T]](newBucket func() B, size int,
	interval time.Duration, opts ...RollingWindowOption[T, B]) *RollingWindow[T, B] {
	if size < 1 {
		panic("size must be greater than 0")
	}

	w := &RollingWindow[T, B]{
		size:     size,
		win:      newWindow[T, B](newBucket, size),
		interval: interval,
		lastTime: timex.Now(),
	}
	for _, opt := range opts {
		opt(w)
	}
	return w
}

// Add adds value to current bucket.
<<<<<<< HEAD

func (rw *RollingWindow) Add(v float64) {
=======
func (rw *RollingWindow[T, B]) Add(v T) {
>>>>>>> f1ed7bd75de44ba1491a2627c36c86e649ae277e
	rw.lock.Lock()
	defer rw.lock.Unlock()
	// 滑动的动作发生在此
	rw.updateOffset()
	rw.win.add(rw.offset, v)
}

// Reduce runs fn on all buckets, ignore current bucket if ignoreCurrent was set.
func (rw *RollingWindow[T, B]) Reduce(fn func(b B)) {
	rw.lock.RLock()
	defer rw.lock.RUnlock()

	var diff int
<<<<<<< HEAD
	span := rw.span() //这么多个桶过期了,原因是时间往前，这段时间没有更新过
	// ignore current bucket, because of partial data
=======
	span := rw.span()
	// ignore the current bucket, because of partial data
>>>>>>> f1ed7bd75de44ba1491a2627c36c86e649ae277e
	if span == 0 && rw.ignoreCurrent {
		diff = rw.size - 1
	} else {
		diff = rw.size - span
	}
	if diff > 0 {
		// [rw.offset - rw.offset+span]之间的桶数据是过期的不应该计入统计.Add和Reduce不能并发，所有这个时间段没有数据写入发生
		offset := (rw.offset + span + 1) % rw.size
		rw.win.reduce(offset, diff, fn)
	}
}

func (rw *RollingWindow[T, B]) span() int {
	offset := int(timex.Since(rw.lastTime) / rw.interval)
	if 0 <= offset && offset < rw.size {
		return offset
	}

	return rw.size
}

func (rw *RollingWindow[T, B]) updateOffset() {
	span := rw.span()
	if span <= 0 {
		return
	}

	offset := rw.offset
	// reset expired buckets
	//既然经过了span个桶的时间没有写入数据
	//那么这些桶内的数据就不应该继续保留了，属于过期数据清空即可
	//可以看到这里全部用的 % 取余操作，可以实现按照下标周期性写入
	//如果超出下标了那就从头开始写，确保新数据一定能够正常写入
	//类似循环数组的效果
	//buck数组从下标由小往大写，时间间隔没有写数据，说明这些桶数据比较老，清空
    // 滑动窗口统计的是duration期间的数据，比如每个250ms的统计数据
	for i := 0; i < span; i++ {
		rw.win.resetBucket((offset + i + 1) % rw.size)
	}

	rw.offset = (offset + span) % rw.size
	now := timex.Now()
	// align to interval time boundary
	// 减掉尾巴
	rw.lastTime = now - (now-rw.lastTime)%rw.interval
}

// Bucket defines the bucket that holds sum and num of additions.
type Bucket[T Numerical] struct {
	Sum   T
	Count int64
}

func (b *Bucket[T]) Add(v T) {
	b.Sum += v
	b.Count++
}

func (b *Bucket[T]) Reset() {
	b.Sum = 0
	b.Count = 0
}

type window[T Numerical, B BucketInterface[T]] struct {
	buckets []B
	size    int
}

func newWindow[T Numerical, B BucketInterface[T]](newBucket func() B, size int) *window[T, B] {
	buckets := make([]B, size)
	for i := 0; i < size; i++ {
		buckets[i] = newBucket()
	}
	return &window[T, B]{
		buckets: buckets,
		size:    size,
	}
}

func (w *window[T, B]) add(offset int, v T) {
	w.buckets[offset%w.size].Add(v)
}

func (w *window[T, B]) reduce(start, count int, fn func(b B)) {
	for i := 0; i < count; i++ {
		fn(w.buckets[(start+i)%w.size])
	}
}

func (w *window[T, B]) resetBucket(offset int) {
	w.buckets[offset%w.size].Reset()
}

// IgnoreCurrentBucket lets the Reduce call ignore current bucket.
<<<<<<< HEAD
// 	某些场景下因为当前正在写入的桶数据并没有经过完整的窗口时间间隔
//	可能导致当前桶的统计并不准确
func IgnoreCurrentBucket() RollingWindowOption {
	return func(w *RollingWindow) {
=======
func IgnoreCurrentBucket[T Numerical, B BucketInterface[T]]() RollingWindowOption[T, B] {
	return func(w *RollingWindow[T, B]) {
>>>>>>> f1ed7bd75de44ba1491a2627c36c86e649ae277e
		w.ignoreCurrent = true
	}
}
