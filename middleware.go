package ginprom

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"math"
	"time"
)

var (
	nop = func(c *gin.Context) {}
)

type Metrics struct {
	TotalRequest     int64
	PrevTotalRequest int64
	Uptime           int64
	QPS              int64 // 当前qps

	// 当前请求总数
	totalRequest *prometheus.CounterVec
	uptime       prometheus.Counter
	qps          prometheus.Gauge
	duration     *prometheus.HistogramVec
}

func NewMetrics(namespace string, registry *prometheus.Registry) *Metrics {
	m := &Metrics{
		// 总请求数
		totalRequest: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: fmt.Sprintf("gin_prom_%s", namespace),
				Name:      "total_request",
				Help:      "Total number of http requests made.",
			}, []string{"status", "path", "method"},
		),

		// gin运行的时间
		uptime: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: fmt.Sprintf("gin_prom_%s", namespace),
				Name:      "uptime",
				Help:      "Gin service uptime.",
			},
		),

		// 每秒的请求 (每60秒更新)
		qps: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: fmt.Sprintf("gin_prom_%s", namespace),
				Name:      "request_per_second",
				Help:      "Request per second.",
			},
		),

		// 请求延迟分布
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: fmt.Sprintf("gin_prom_%s", namespace),
			Name:      "requests_duration_seconds",
			Help:      "Request latencies in seconds.",
			Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.250, 0.5, 1},
		}, []string{"status", "path", "method"}),
	}

	registry.MustRegister(m.totalRequest, m.uptime, m.qps, m.duration)

	// 计算和更新
	go func() {
		for {
			time.Sleep(time.Second)

			m.Uptime++
			m.uptime.Inc()

			if m.Uptime%3 == 0 {
				// 每分钟的qps
				if m.PrevTotalRequest == 0 {
					m.PrevTotalRequest = m.TotalRequest
					continue
				}
				m.QPS = int64(math.Ceil(float64(m.TotalRequest-m.PrevTotalRequest) / 3))

				m.qps.Set(float64(m.QPS))
				m.PrevTotalRequest = m.TotalRequest
			}

		}
	}()

	return m
}

/*
AddMetrics gin的中间件
namespace 代表接口统计所属的命名空间，中间件可以统计不同的接口或者端口
registry 代表 prometheus的 registry
*/
func Observe(metrics *Metrics) gin.HandlerFunc {
	if metrics == nil {
		return nop
	}

	// 统计
	return func(c *gin.Context) {
		startTime := time.Now()
		c.Next()

		labels := []string{fmt.Sprintf("%d", c.Writer.Status()), c.Request.URL.Path, c.Request.Method}

		metrics.TotalRequest++
		metrics.totalRequest.WithLabelValues(labels...).Inc()
		metrics.duration.WithLabelValues(labels...).Observe(time.Since(startTime).Seconds())
	}
}
