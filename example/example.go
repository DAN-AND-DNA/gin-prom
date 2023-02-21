package main

import (
	ginprom "github.com/dan-and-dna/gin-prom"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"log"
	"time"
)

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	reg := prometheus.NewRegistry()

	metrics := ginprom.NewMetrics("default", reg)

	r.Use(ginprom.Export(metrics))

	r.GET("/ping", func(c *gin.Context) {
		time.Sleep(2 * time.Second)
		c.JSON(200, gin.H{"message": "ok"})
	})

	/*
		go func() {
			for {
				// 30 qps
				time.Sleep(30 * time.Millisecond)
				w := httptest.NewRecorder()
				req, _ := http.NewRequest("GET", "/ping", nil)
				r.ServeHTTP(w, req)
			}
		}()

	*/

	go func() {
		for {
			time.Sleep(5 * time.Second)

			/*
				metric := &dto.Metric{}
				o, err := metrics.PromDuration.MetricVec.GetMetricWithLabelValues([]string{"200", "/ping", "GET"}...)
				if err != nil {
					panic(err)
				}

				o.Write(metric)
				fmt.Println(proto.MarshalTextString(metric))

			*/
			log.Printf("total_request: %d qps: %d \n", metrics.TotalRequest, metrics.QPS)
		}
	}()

	_ = r.Run() // 监听并在 0.0.0.0:8080 上启动服务
}
