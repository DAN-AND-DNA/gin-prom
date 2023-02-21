package main

import (
	ginprom "github.com/dan-and-dna/gin-prom"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"log"
	"net/http"
	"net/http/httptest"
	"time"
)

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	reg := prometheus.NewRegistry()

	metrics := ginprom.NewMetrics("default", reg)

	r.Use(ginprom.AddMetrics(metrics))

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "ok"})
	})

	go func() {
		for {
			// 30 qps
			time.Sleep(30 * time.Millisecond)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/ping", nil)
			r.ServeHTTP(w, req)
		}
	}()

	go func() {
		for {
			time.Sleep(5 * time.Second)
			log.Printf("total_request: %d qps: %d \n", metrics.TotalRequest, metrics.QPS)
		}
	}()

	_ = r.Run() // 监听并在 0.0.0.0:8080 上启动服务
}
