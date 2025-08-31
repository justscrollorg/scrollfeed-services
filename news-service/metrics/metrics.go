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

	// Business metrics for news service
	NewsArticlesFetched = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "news_articles_fetched_total",
			Help: "Total number of news articles fetched",
		},
		[]string{"source", "status"},
	)

	NewsArticlesServed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "news_articles_served_total",
			Help: "Total number of news articles served to clients",
		},
		[]string{"category", "source"},
	)

	NewsStreamConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "news_stream_connections_active",
			Help: "Number of active news stream connections",
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

	// NATS metrics
	NatsMessagesPublished = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nats_messages_published_total",
			Help: "Total number of NATS messages published",
		},
		[]string{"subject", "status"},
	)

	NatsMessagesReceived = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nats_messages_received_total",
			Help: "Total number of NATS messages received",
		},
		[]string{"subject", "status"},
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
