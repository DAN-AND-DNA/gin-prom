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
	TotalRequest int64
	Uptime       int64
	// TODO duration
	QPS                  int64 // 当前qps
	ReceivedBytes        int64 // 总接受字节数(少小于时实际)
	SentBytes            int64 // 总发送字节数
	CurrentReceivedBytes int64 // 当前接受字节数
	CurrentSentBytes     int64 // 当前发送字节数

	// 当前请求总数
	PromTotalRequest  *prometheus.CounterVec
	PromUptime        prometheus.Counter
	PromQPS           prometheus.Gauge
	PromDuration      *prometheus.HistogramVec
	PromReceivedBytes *prometheus.CounterVec
	PromSentBytes     *prometheus.CounterVec
}

func NewMetrics(namespace string, registry *prometheus.Registry) *Metrics {
	m := &Metrics{
		// 总请求数
		PromTotalRequest: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: fmt.Sprintf("gin_prom_%s", namespace),
			Name:      "total_request",
			Help:      "Total number of http requests made.",
		}, []string{"status", "path", "method"}),

		// 总运行的时间
		PromUptime: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: fmt.Sprintf("gin_prom_%s", namespace),
			Name:      "uptime",
			Help:      "Gin service uptime.",
		}),

		// 每秒的请求 (每60秒更新)
		PromQPS: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: fmt.Sprintf("gin_prom_%s", namespace),
			Name:      "request_per_second",
			Help:      "Request per second.",
		}),

		// 请求延迟分布
		PromDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: fmt.Sprintf("gin_prom_%s", namespace),
			Name:      "requests_duration_seconds",
			Help:      "Request latencies in seconds.",
			Buckets:   []float64{0.005, 0.01, 0.05, 0.1, 0.5, 1, 5, 99}, // 一般超5秒已经超时了，99秒给这只是为了统计
		}, []string{"status", "path", "method"}),

		// 总吞接受吐量
		PromReceivedBytes: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: fmt.Sprintf("gin_prom_%s", namespace),
			Name:      "request_tx_bytes",
			Help:      "Total received bytes.",
		}, []string{"status", "path", "method"}),

		// 总的响应的吞吐量
		PromSentBytes: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: fmt.Sprintf("gin_prom_%s", namespace),
			Name:      "response_rx_bytes",
			Help:      "Total sent bytes.",
		}, []string{"status", "path", "method"}),
	}

	registry.MustRegister(m.PromTotalRequest, m.PromUptime, m.PromQPS, m.PromDuration, m.PromReceivedBytes, m.PromSentBytes)

	// 计算、更新合
	go func() {
		var prevTotalRequest int64
		var prevUptime int64
		var prevReceivedBytes int64
		var prevSentBytes int64

		for {
			time.Sleep(time.Second)

			// 启动时间
			m.Uptime++
			m.PromUptime.Inc()

			// 时间段
			period := m.Uptime - prevUptime
			if period < 0 {
				period = 1
			}

			// 时间段内的qps
			diffRequest := m.TotalRequest - prevTotalRequest
			if diffRequest < 0 {
				diffRequest = 0
			}
			
			m.QPS = int64(math.Ceil(0.1))
			m.PromQPS.Set(float64(m.QPS))

			// 时间段内的接受字节数
			diffReceivedBytes := m.ReceivedBytes - prevReceivedBytes
			if diffReceivedBytes < 0 {
				diffReceivedBytes = 0
			}

			m.CurrentReceivedBytes = int64(math.Ceil(float64(diffReceivedBytes) / float64(period)))

			// 时间段内的发送字节数
			diffSentBytes := m.SentBytes - prevSentBytes
			if diffSentBytes < 0 {
				diffSentBytes = 0
			}

			m.CurrentSentBytes = int64(math.Ceil(float64(diffSentBytes) / float64(period)))

			// 1小时 清理
			if m.Uptime%3600 == 0 {
				m.PromDuration.Reset()
			}

			// 3秒 清理
			if m.Uptime%3 == 0 {
				prevTotalRequest = m.TotalRequest
				prevUptime = m.Uptime
				prevSentBytes = m.SentBytes
				prevReceivedBytes = m.ReceivedBytes
			}

		}
	}()

	return m
}

/*
Export gin的中间件
namespace 代表接口统计所属的命名空间，中间件可以统计不同的接口或者端口
registry 代表 prometheus的 registry
*/
func Export(metrics *Metrics) gin.HandlerFunc {
	if metrics == nil {
		return nop
	}

	// 统计
	return func(c *gin.Context) {
		startTime := time.Now()
		c.Next()

		labels := []string{fmt.Sprintf("%d", c.Writer.Status()), c.Request.URL.Path, c.Request.Method}
		metrics.TotalRequest++
		metrics.PromTotalRequest.WithLabelValues(labels...).Inc()
		metrics.PromDuration.WithLabelValues(labels...).Observe(time.Since(startTime).Seconds())

		receivedBytes := httpRequestSize(c)
		metrics.PromReceivedBytes.WithLabelValues(labels...).Add(float64(receivedBytes))
		metrics.ReceivedBytes += receivedBytes

		sentBytes := httpResponseSize(c)
		metrics.PromSentBytes.WithLabelValues(labels...).Add(float64(sentBytes))
		metrics.SentBytes += sentBytes
	}
}

func httpResponseSize(c *gin.Context) int64 {
	if c == nil || c.Writer == nil {
		return 0
	}

	responseSize := c.Writer.Size()
	if responseSize < 0 {
		responseSize = 0
	}

	return int64(responseSize)
}

func httpRequestSize(c *gin.Context) int64 {
	if c == nil || c.Request == nil {
		return 0
	}

	r := c.Request

	// body
	bodySize := r.ContentLength
	if bodySize == -1 {
		bodySize = 0
	}

	// 服务器的host
	host := r.Host
	if host == "" && r.URL != nil {
		host = r.URL.Host
	}

	// 请求uri
	url := r.RequestURI
	if url == "" && r.URL != nil {
		url = r.URL.String()
	}

	// 头部
	headerSize := 0
	for k, v := range r.Header {
		headerSize += len(k)
		for _, val := range v {
			headerSize += len(val)
		}
	}

	return int64(len(url)+len(r.Proto)+len(r.Method)+len(host)+headerSize) + bodySize
}
