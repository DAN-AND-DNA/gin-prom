package ginprom

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"sync"
	"time"
)

var (
	namespaces map[string]*Metrics = make(map[string]*Metrics)
	mu         sync.RWMutex
)

type Metrics struct {
	TotalRequest int64
	Uptime       int64
}

func GinProm(namespace string, registry *prometheus.Registry) gin.HandlerFunc {
	// 验证参数
	var metrics *Metrics
	func() {
		if registry == nil {
			panic("registry is nil")
		}

		mu.Lock()
		defer mu.Unlock()
		if _, ok := namespaces[namespace]; ok {
			panic("namespace cannot be duplicated")
		} else {
			metrics = &Metrics{}
			namespaces[namespace] = metrics
		}
	}()

	// 当前请求总数
	totalRequest := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "total_request",
			Help:      "Total number of http requests made.",
		}, []string{"status", "endpoint", "method"},
	)

	registry.MustRegister(totalRequest)

	// 统计
	return func(c *gin.Context) {
		metrics.TotalRequest++
		totalRequest.WithLabelValues("200", c.Request.URL.Path, c.Request.Method).Inc()
		start := time.Now()
		_ = start
	}
}
