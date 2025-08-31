package middleware

import (
	"analytics-service/metrics"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// PrometheusMiddleware creates a middleware for collecting Prometheus metrics
func PrometheusMiddleware(serviceName string) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		// Process request
		c.Next()

		// Collect metrics
		duration := time.Since(start).Seconds()
		statusCode := strconv.Itoa(c.Writer.Status())

		// Record metrics
		metrics.HttpRequestsTotal.WithLabelValues(
			method,
			path,
			statusCode,
			serviceName,
		).Inc()

		metrics.HttpRequestDuration.WithLabelValues(
			method,
			path,
			serviceName,
		).Observe(duration)
	})
}
