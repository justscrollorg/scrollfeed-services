package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP request metrics
	HttpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status_code", "service"},
	)

	HttpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "service"},
	)

	// Business metrics for analytics service
	AnalyticsEventsTracked = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "analytics_events_tracked_total",
			Help: "Total number of analytics events tracked",
		},
		[]string{"event_type", "source"},
	)

	AnalyticsStatsRequests = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "analytics_stats_requests_total",
			Help: "Total number of analytics stats requests",
		},
	)

	// Database metrics
	MongoOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mongo_operations_total",
			Help: "Total number of MongoDB operations",
		},
		[]string{"operation", "collection", "status"},
	)

	MongoOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mongo_operation_duration_seconds",
			Help:    "MongoDB operation duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation", "collection"},
	)

	// Application health metrics
	ApplicationInfo = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "application_info",
			Help: "Application information",
		},
		[]string{"service", "version", "environment"},
	)

	// Active connections and resources
	ActiveConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "active_connections",
			Help: "Number of active connections",
		},
	)
)

// Initialize metrics with default values
func Init(serviceName, version, environment string) {
	ApplicationInfo.WithLabelValues(serviceName, version, environment).Set(1)
}
