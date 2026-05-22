package middleware

import (
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type Metrics struct {
	mu             sync.Mutex
	RequestCount   map[string]int64
	RequestLatency map[string][]time.Duration
	StatusCodes    map[int]int64
	StartTime      time.Time
}

var globalMetrics = &Metrics{
	RequestCount:   make(map[string]int64),
	RequestLatency: make(map[string][]time.Duration),
	StatusCodes:    make(map[int]int64),
	StartTime:      time.Now(),
}

func MetricsCollector() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)

		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		globalMetrics.mu.Lock()
		globalMetrics.RequestCount[path]++
		globalMetrics.StatusCodes[c.Writer.Status()]++
		latencies := globalMetrics.RequestLatency[path]
		if len(latencies) > 100 {
			latencies = latencies[1:]
		}
		globalMetrics.RequestLatency[path] = append(latencies, duration)
		globalMetrics.mu.Unlock()
	}
}

func MetricsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		globalMetrics.mu.Lock()
		defer globalMetrics.mu.Unlock()

		var lines []string
		lines = append(lines, "# HELP postoffice_uptime_seconds Time since server start")
		lines = append(lines, "# TYPE postoffice_uptime_seconds gauge")
		lines = append(lines, "postoffice_uptime_seconds "+strconv.FormatFloat(time.Since(globalMetrics.StartTime).Seconds(), 'f', 1, 64))

		lines = append(lines, "# HELP postoffice_http_requests_total Total HTTP requests")
		lines = append(lines, "# TYPE postoffice_http_requests_total counter")
		for path, count := range globalMetrics.RequestCount {
			lines = append(lines, "postoffice_http_requests_total{path=\""+path+"\"} "+strconv.FormatInt(count, 10))
		}

		lines = append(lines, "# HELP postoffice_http_status_total HTTP responses by status code")
		lines = append(lines, "# TYPE postoffice_http_status_total counter")
		for code, count := range globalMetrics.StatusCodes {
			lines = append(lines, "postoffice_http_status_total{code=\""+strconv.Itoa(code)+"\"} "+strconv.FormatInt(count, 10))
		}

		output := ""
		for _, l := range lines {
			output += l + "\n"
		}
		c.Data(200, "text/plain; charset=utf-8", []byte(output))
	}
}
