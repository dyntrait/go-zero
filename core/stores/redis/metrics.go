package redis

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	red "github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/metric"
)

// 信息的采集埋点在hook中
const namespace = "redis_client"

var (
	metricReqDur = metric.NewHistogramVec(&metric.HistogramVecOpts{
		Namespace: namespace,
		Subsystem: "requests",
		Name:      "duration_ms",
		Help:      "redis client requests duration(ms).",
		Labels:    []string{"command"},
		Buckets:   []float64{0.25, 0.5, 1, 1.5, 2, 3, 5, 10, 25, 50, 100, 250, 500, 1000, 2000, 5000, 10000, 15000},
	})
	metricReqErr = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: namespace,
		Subsystem: "requests",
		Name:      "error_total",
		Help:      "redis client requests error count.",
		Labels:    []string{"command", "error"},
	})
	metricSlowCount = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: namespace,
		Subsystem: "requests",
		Name:      "slow_total",
		Help:      "redis client requests slow count.",
		Labels:    []string{"command"},
	})

	connLabels                         = []string{"key", "client_type"} //key是redis的地址
	connCollector                      = newCollector()                 //收集像redis连接这类信息
	_             prometheus.Collector = (*collector)(nil)
)

type (
	statGetter struct {
		clientType string
		key        string //key是要连接的redis地址
		poolSize   int
		poolStats  func() *red.PoolStats //调用这个函数可以获取redis连接上的一些统计信息
	}

	// collector collects statistics from a redis client.
	// It implements the prometheus.Collector interface.
	collector struct {
		hitDesc     *prometheus.Desc
		missDesc    *prometheus.Desc
		timeoutDesc *prometheus.Desc
		totalDesc   *prometheus.Desc
		idleDesc    *prometheus.Desc
		staleDesc   *prometheus.Desc
		maxDesc     *prometheus.Desc

		clients []*statGetter //有多少个redis连接客户端，在建立redis连接时赋值
		lock    sync.Mutex
	}
)

func newCollector() *collector {
	c := &collector{
		hitDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "pool_hit_total"),
			"Number of times a connection was found in the pool",
			connLabels, nil,
		),
		missDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "pool_miss_total"),
			"Number of times a connection was not found in the pool",
			connLabels, nil,
		),
		timeoutDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "pool_timeout_total"),
			"Number of times a timeout occurred when looking for a connection in the pool",
			connLabels, nil,
		),
		totalDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "pool_conn_total_current"),
			"Current number of connections in the pool",
			connLabels, nil,
		),
		idleDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "pool_conn_idle_current"),
			"Current number of idle connections in the pool",
			connLabels, nil,
		),
		staleDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "pool_conn_stale_total"),
			"Number of times a connection was removed from the pool because it was stale",
			connLabels, nil,
		),
		maxDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "pool_conn_max"),
			"Max number of connections in the pool",
			connLabels, nil,
		),
	}
	//任何实现了 prometheus.Collector 接口的对象都可以被注册到 Prometheus 注册表中，以便自动收集和导出指标数据。
	//常见的实现包括 prometheus.Gauge, prometheus.Counter, prometheus.Histogram 和 prometheus.Summary 等类型，
	//这些类型分别代表不同种类的指标数据

	prometheus.MustRegister(c)

	return c
}

// Describe implements the prometheus.Collector interface.
// 此方法用于发送指标描述符到提供的通道。描述符包含了 Prometheus 指标的名称、帮助信息、标签等。
func (s *collector) Describe(descs chan<- *prometheus.Desc) {
	descs <- s.hitDesc
	descs <- s.missDesc
	descs <- s.timeoutDesc
	descs <- s.totalDesc
	descs <- s.idleDesc
	descs <- s.staleDesc
	descs <- s.maxDesc
}

// Collect implements the prometheus.Collector interface.
// 此方法用于收集指标值，并将它们发送到提供的通道。
func (s *collector) Collect(metrics chan<- prometheus.Metric) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for _, client := range s.clients {
		key, clientType := client.key, client.clientType // key是rdis的地址
		stats := client.poolStats()

		metrics <- prometheus.MustNewConstMetric(
			s.hitDesc,
			prometheus.CounterValue,
			float64(stats.Hits),
			key,
			clientType,
		)
		metrics <- prometheus.MustNewConstMetric(
			s.missDesc,
			prometheus.CounterValue,
			float64(stats.Misses),
			key,
			clientType,
		)
		metrics <- prometheus.MustNewConstMetric(
			s.timeoutDesc,
			prometheus.CounterValue,
			float64(stats.Timeouts),
			key,
			clientType,
		)
		metrics <- prometheus.MustNewConstMetric(
			s.totalDesc,
			prometheus.GaugeValue,
			float64(stats.TotalConns),
			key,
			clientType,
		)
		metrics <- prometheus.MustNewConstMetric(
			s.idleDesc,
			prometheus.GaugeValue,
			float64(stats.IdleConns),
			key,
			clientType,
		)
		metrics <- prometheus.MustNewConstMetric(
			s.staleDesc,
			prometheus.CounterValue,
			float64(stats.StaleConns),
			key,
			clientType,
		)
		metrics <- prometheus.MustNewConstMetric(
			s.maxDesc,
			prometheus.CounterValue,
			float64(client.poolSize),
			key,
			clientType,
		)
	}
}

func (s *collector) registerClient(client *statGetter) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.clients = append(s.clients, client)
}
