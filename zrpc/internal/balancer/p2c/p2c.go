package p2c

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/syncx"
	"github.com/zeromicro/go-zero/core/timex"
	"github.com/zeromicro/go-zero/zrpc/internal/codes"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/resolver"
)

const (
	// Name is the name of p2c balancer.
	Name = "p2c_ewma" //峰值 EWMA（预测）策略
	// 指数移动加权平均法（Exponential Moving Average，EMA）是一种用于平滑时间序列数据的统计方法。
	//它是移动平均法的一种改进版本，对最新的观测值给予更大的权重，而较旧的观测值则逐渐减小其权重。

	decayTime       = int64(time.Second * 10) // default value from finagle
	forcePick       = int64(time.Second)
	initSuccess     = 1000
	throttleSuccess = initSuccess / 2
	penalty         = int64(math.MaxInt32) //惩罚
	pickTimes       = 3
	logInterval     = time.Minute
)

var emptyPickResult balancer.PickResult

func init() {
	//新生成一个grpc.base.baseBuilder实例，注册到grpc的全局Builder map集合里 。
	// baseBuilder有一个pickerBuilder成员，调用pickerBuilder的Build()可以得到balancer.Picker
	//invoke时调用picker的pick选择一个conn，把请求发出去
	//其实所有http2的连接由balancer管理，可以认为balancer是一个连接池
	//而picker就是从池子里挑出一个最优先的连接，把请求发出去
	//grpc.Dial时:
	//         1.resolver获取后端服务的地址列表，调用balancer建立一些连接
	//         2.resolver监控到后端服务上线或者上线时，通知balancer创建新连接或者移除旧的连接

	//当grpc.Invoke时，会使用pickerBuilder的Build()可以得到balancer.Picker，然后使用balancer.Picker()的Pick挑选一个最优的连接

	balancer.Register(newBuilder())
}

type p2cPickerBuilder struct{}

// gRPC 在节点有更新的时候会调用 Build 方法，传入所有节点信息，
// 我们在这里把每个节点信息用 subConn 结构保存起来。并归并到一起用 p2cPicker 结构保存起来
//
//	A SubConn represents a single connection to a gRPC backend service.
func (b *p2cPickerBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	readySCs := info.ReadySCs
	if len(readySCs) == 0 {
		return base.NewErrPicker(balancer.ErrNoSubConnAvailable)
	}

	// 收集connectivity.Ready 状态供picker挑选
	var conns []*subConn
	for conn, connInfo := range readySCs {
		conns = append(conns, &subConn{
			addr:    connInfo.Address,
			conn:    conn,
			success: initSuccess,
		})
	}

	return &p2cPicker{
		conns: conns,
		r:     rand.New(rand.NewSource(time.Now().UnixNano())),
		stamp: syncx.NewAtomicDuration(),
	}
}

func newBuilder() balancer.Builder {
	return base.NewBalancerBuilder(Name, new(p2cPickerBuilder), base.Config{HealthCheck: true})
}

type p2cPicker struct {
	conns []*subConn
	r     *rand.Rand
	stamp *syncx.AtomicDuration //日志输出的间隔
	lock  sync.Mutex
}

//	gRPC 是如何选择一个连接进行请求的发送？
//
// 当 gRPC 客户端发起调用的时候，会调用 ClientConn 的 Invoke 方法
// 最终会调用 p2cPicker.Pick 方法获取连接，我们自定义的负载均衡算法一般都在 Pick 方法中实现，获取到连接之后，通过 sendMsg 发送请求
// EWMA 指数移动加权平均的算法，表示是一段时间内的均值。该算法相对于算数平均来说对于突然的网络抖动没有那么敏感，
// 突然的抖动不会体现在请求的 lag 中，从而可以让算法更加均衡
func (p *p2cPicker) Pick(_ balancer.PickInfo) (balancer.PickResult, error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	var chosen *subConn
	switch len(p.conns) {
	case 0:
		return emptyPickResult, balancer.ErrNoSubConnAvailable
	case 1:
		chosen = p.choose(p.conns[0], nil)
	case 2:
		chosen = p.choose(p.conns[0], p.conns[1])
	default:
		var node1, node2 *subConn
		for i := 0; i < pickTimes; i++ {
			a := p.r.Intn(len(p.conns))
			b := p.r.Intn(len(p.conns) - 1)
			if b >= a {
				b++
			}
			node1 = p.conns[a]
			node2 = p.conns[b]
			// 如果这次选择的节点达到了健康要求, 就中断选择
			if node1.healthy() && node2.healthy() {
				break
			}
		}

		chosen = p.choose(node1, node2) //选出最优的subConn
	}

	atomic.AddInt64(&chosen.inflight, 1)
	atomic.AddInt64(&chosen.requests, 1)

	return balancer.PickResult{
		SubConn: chosen.conn,             // SubConn is the connection to use for this pick, if its state is Ready.
		Done:    p.buildDoneFunc(chosen), // Done is called when the RPC is completed，这是一个函数
	}, nil
}

// 某个subConn被选中了，这个subConn发送完了请求后的回调函数
// 根据请求来回之间的延迟时间，更新被选中连接的lag,和success.相当于告诉其他连接，我这次又被选中了

func (p *p2cPicker) buildDoneFunc(c *subConn) func(info balancer.DoneInfo) {
	start := int64(timex.Now())
	return func(info balancer.DoneInfo) {
		atomic.AddInt64(&c.inflight, -1)
		now := timex.Now()
		// 保存本次请求结束时的时间点，并取出上次请求时的时间点
		last := atomic.SwapInt64(&c.last, int64(now))

		td := int64(now) - last //两次请求的间隔越大
		if td < 0 {
			td = 0
		}
		// 系数 w 是一个时间衰减值，即两次请求的间隔越大，则系数 w 就越小。
		// 用牛顿冷却定律中的衰减函数模型计算EWMA算法中的β值=1/(e的(k*(2次请求间隔）)
		//从数学的角度来看，这个表达式可能用于计算某种基于时间的衰减或增长因子。例如，在物理或工程应用中，当某个量随时间呈指数衰减时，你可能会看到这样的表达式。
		// 具体地，这个表达式的值将是一个介于0和正无穷大之间的数（除非decayTime为0，这将导致除以0的错误）。
		//如果-td / decayTime是一个负数，那么math.Exp(...)的值将在0和1之间；如果它是一个正数，那么值将大于1
		w := math.Exp(float64(-td) / float64(decayTime)) //计算e的w次方
		// 保存本次请求的耗时
		lag := int64(now) - start
		if lag < 0 {
			lag = 0
		}
		olag := atomic.LoadUint64(&c.lag)
		if olag == 0 {
			w = 0
		}
		// 对最新的观测值给予更大的权重，而较旧的观测值则逐渐减小其权重。
		atomic.StoreUint64(&c.lag, uint64(float64(olag)*w+float64(lag)*(1-w)))
		success := initSuccess
		if info.Err != nil && !codes.Acceptable(info.Err) {
			success = 0
		}
		osucc := atomic.LoadUint64(&c.success)
		// 对最新的观测值给予更大的权重，而较旧的观测值则逐渐减小其权重。
		atomic.StoreUint64(&c.success, uint64(float64(osucc)*w+float64(success)*(1-w)))

		stamp := p.stamp.Load()
		if now-stamp >= logInterval {
			if p.stamp.CompareAndSwap(stamp, now) {
				p.logStats()
			}
		}
	}
}

func (p *p2cPicker) choose(c1, c2 *subConn) *subConn {
	start := int64(timex.Now())
	if c2 == nil {
		atomic.StoreInt64(&c1.pick, start)
		return c1
	}

	if c1.load() > c2.load() {
		c1, c2 = c2, c1
	}

	pick := atomic.LoadInt64(&c2.pick)
	if start-pick > forcePick && atomic.CompareAndSwapInt64(&c2.pick, pick, start) {
		return c2
	}

	atomic.StoreInt64(&c1.pick, start)
	return c1
}

func (p *p2cPicker) logStats() {
	var stats []string

	p.lock.Lock()
	defer p.lock.Unlock()

	for _, conn := range p.conns {
		stats = append(stats, fmt.Sprintf("conn: %s, load: %d, reqs: %d",
			conn.addr.Addr, conn.load(), atomic.SwapInt64(&conn.requests, 0)))
	}

	logx.Statf("p2c - %s", strings.Join(stats, "; "))
}

type subConn struct {
	lag      uint64 //连接的请求延迟 lag 用来保存 ewma 值
	inflight int64  // 用在保存当前节点正在处理的请求总数
	success  uint64 // 用来标识一段时间内此连接的健康状态
	requests int64  // 用来保存请求总数
	last     int64  //  用来保存上一次请求耗时,
	pick     int64  //chose time 保存上一次被选中的时间点
	addr     resolver.Address
	conn     balancer.SubConn
}

func (c *subConn) healthy() bool {
	return atomic.LoadUint64(&c.success) > throttleSuccess
}

//节点的 load 值是通过该连接的请求延迟 lag 和当前请求数 inflight 的乘积所得，如果请求的延迟越大或者当前正在处理的请求数越多表明该节点的负载越高。

// ewma 相当于平均请求耗时，inflight 是当前节点正在处理请求的数量，相乘大致计算出了当前节点的网络负载。还需要多少时间才能处理完连接上已有的请求
func (c *subConn) load() int64 {
	// plus one to avoid multiply zero
	lag := int64(math.Sqrt(float64(atomic.LoadUint64(&c.lag) + 1)))
	load := lag * (atomic.LoadInt64(&c.inflight) + 1)
	if load == 0 {
		return penalty
	}

	return load
}
