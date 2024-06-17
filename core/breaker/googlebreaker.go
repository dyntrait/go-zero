package breaker

import (
	"time"

	"github.com/zeromicro/go-zero/core/collection"
	"github.com/zeromicro/go-zero/core/mathx"
	"github.com/zeromicro/go-zero/core/syncx"
	"github.com/zeromicro/go-zero/core/timex"
)

const (
	// 250ms for bucket duration
	window            = time.Second * 10
	buckets           = 40
	forcePassDuration = time.Second
	k                 = 1.5
	minK              = 1.1
	protection        = 5
)

// googleBreaker is a netflixBreaker pattern from google.
// see Client-Side Throttling section in https://landing.google.com/sre/sre-book/chapters/handling-overload/
<<<<<<< HEAD
type googleBreaker struct {
	k     float64 //敏感度，go-zero中默认值为1.5
	stat  *collection.RollingWindow//滑动窗口，用于记录最近一段时间内的请求总数，成功总数
	proba *mathx.Proba //随机产生0.0-1.0之间的双精度浮点数
}

func newGoogleBreaker() *googleBreaker {
	bucketDuration := time.Duration(int64(window) / int64(buckets)) //10s40个桶
	st := collection.NewRollingWindow(buckets, bucketDuration)
=======
type (
	googleBreaker struct {
		k        float64
		stat     *collection.RollingWindow[int64, *bucket]
		proba    *mathx.Proba
		lastPass *syncx.AtomicDuration
	}

	windowResult struct {
		accepts        int64
		total          int64
		failingBuckets int64
		workingBuckets int64
	}
)

func newGoogleBreaker() *googleBreaker {
	bucketDuration := time.Duration(int64(window) / int64(buckets))
	st := collection.NewRollingWindow[int64, *bucket](func() *bucket {
		return new(bucket)
	}, buckets, bucketDuration)
>>>>>>> f1ed7bd75de44ba1491a2627c36c86e649ae277e
	return &googleBreaker{
		stat:     st,
		k:        k,
		proba:    mathx.NewProba(),
		lastPass: syncx.NewAtomicDuration(),
	}
}

// 计算丢弃请求的概率
// 1.requests： 窗口时间内的请求总数
// 2. accepts：正常请求数量
// 3. K：敏感度，K 越小越容易丢请求，一般推荐 1.5-2 之间
// 算法解释：
//1. 正常情况下 requests=accepts，所以概率是 0。
//2. 随着正常请求数量减少，当达到 requests == K* accepts 继续请求时，概率 P 会逐渐比 0 大开始按照概率逐渐丢弃一些请求，如果故障严重则丢包会越来越多，假如窗口时间内 accepts==0 则完全熔断。
//3. 当应用逐渐恢复正常时，accepts、requests 同时都在增加，但是 K*accepts 会比 requests 增加的更快，所以概率很快就会归 0，关闭熔断。
func (b *googleBreaker) accept() error {
	var w float64
	history := b.history()
	w = b.k - (b.k-minK)*float64(history.failingBuckets)/buckets
	weightedAccepts := mathx.AtLeast(w, minK) * float64(history.accepts)
	// https://landing.google.com/sre/sre-book/chapters/handling-overload/#eq2101
	// for better performance, no need to care about the negative ratio
	dropRatio := (float64(history.total-protection) - weightedAccepts) / float64(history.total+1)
	if dropRatio <= 0 {
		return nil
	}

	lastPass := b.lastPass.Load()
	if lastPass > 0 && timex.Since(lastPass) > forcePassDuration {
		b.lastPass.Set(timex.Now())
		return nil
	}

	dropRatio *= float64(buckets-history.workingBuckets) / buckets

	if b.proba.TrueOnProba(dropRatio) {
		return ErrServiceUnavailable
	}

	b.lastPass.Set(timex.Now())

	return nil
}

func (b *googleBreaker) allow() (internalPromise, error) {
	if err := b.accept(); err != nil {
		b.markDrop()
		return nil, err
	}

	return googlePromise{
		b: b,
	}, nil
}

func (b *googleBreaker) doReq(req func() error, fallback Fallback, acceptable Acceptable) error {
	if err := b.accept(); err != nil {
		b.markDrop()
		if fallback != nil {
			return fallback(err)
		}

		return err
	}

	var succ bool
	defer func() {
		// if req() panic, success is false, mark as failure
		if succ {
			b.markSuccess()
		} else {
			b.markFailure()
		}
	}()

	err := req()
	if acceptable(err) {
		succ = true
	}

	return err
}

func (b *googleBreaker) markDrop() {
	b.stat.Add(drop)
}

func (b *googleBreaker) markFailure() {
	b.stat.Add(fail)
}

func (b *googleBreaker) markSuccess() {
	b.stat.Add(success)
}

func (b *googleBreaker) history() windowResult {
	var result windowResult

	b.stat.Reduce(func(b *bucket) {
		result.accepts += b.Success
		result.total += b.Sum
		if b.Failure > 0 {
			result.workingBuckets = 0
		} else if b.Success > 0 {
			result.workingBuckets++
		}
		if b.Success > 0 {
			result.failingBuckets = 0
		} else if b.Failure > 0 {
			result.failingBuckets++
		}
	})

	return result
}

type googlePromise struct {
	b *googleBreaker
}

func (p googlePromise) Accept() {
	p.b.markSuccess()
}

func (p googlePromise) Reject() {
	p.b.markFailure()
}
