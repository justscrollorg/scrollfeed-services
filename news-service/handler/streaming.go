package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"news-service/model"
	"time"

	"github.com/nats-io/nats.go"
)

// StreamingConfig holds configuration for NATS JetStream
type StreamingConfig struct {
	URL            string
	StreamName     string
	Subjects       []string
	MaxAge         time.Duration
	MaxBytes       int64
	MaxMsgs        int64
	Replicas       int
	ConsumerConfig ConsumerConfig
}

// ConsumerConfig holds consumer configuration
type ConsumerConfig struct {
	DurableName    string
	DeliverSubject string
	AckPolicy      nats.AckPolicy
	MaxDeliver     int
	AckWait        time.Duration
	ReplayPolicy   nats.ReplayPolicy
}

// NATSStreamingService handles JetStream operations
type NATSStreamingService struct {
	nc      *nats.Conn
	js      nats.JetStreamContext
	config  *StreamingConfig
	streams map[string]*StreamInfo
}

// StreamInfo holds information about a stream
type StreamInfo struct {
	Name     string
	Subjects []string
	Consumer string
}

// NewsEvent represents different types of news events
type NewsEvent struct {
	Type      string    `json:"type"` // "article_published", "article_updated", "trending_topic", "analytics"
	Timestamp time.Time `json:"timestamp"`
	Source    string    `json:"source"`
	Region    string    `json:"region"`
	Data      EventData `json:"data"`
}

// EventData is a union type for different event data
type EventData struct {
	Article   *model.Article `json:"article,omitempty"`
	Analytics *AnalyticsData `json:"analytics,omitempty"`
	Trending  *TrendingData  `json:"trending,omitempty"`
	Metrics   *MetricsData   `json:"metrics,omitempty"`
}

// AnalyticsData represents analytics events
type AnalyticsData struct {
	ArticleID      string           `json:"article_id"`
	ViewCount      int64            `json:"view_count"`
	ShareCount     int64            `json:"share_count"`
	EngagementRate float64          `json:"engagement_rate"`
	Demographics   map[string]int64 `json:"demographics"`
	Tags           []string         `json:"tags"`
}

// TrendingData represents trending topic events
type TrendingData struct {
	Topic     string   `json:"topic"`
	Score     float64  `json:"score"`
	Articles  []string `json:"articles"`
	Keywords  []string `json:"keywords"`
	TrendType string   `json:"trend_type"` // "rising", "peak", "declining"
}

// MetricsData represents system metrics events
type MetricsData struct {
	ServiceName   string                 `json:"service_name"`
	RequestCount  int64                  `json:"request_count"`
	ErrorRate     float64                `json:"error_rate"`
	ResponseTime  time.Duration          `json:"response_time"`
	CustomMetrics map[string]interface{} `json:"custom_metrics"`
}

// NewNATSStreamingService creates a new streaming service with JetStream
func NewNATSStreamingService(config *StreamingConfig) (*NATSStreamingService, error) {
	// Connect to NATS
	nc, err := nats.Connect(config.URL,
		nats.Name("news-service-streaming"),
		nats.ReconnectWait(2*time.Second),
		nats.MaxReconnects(-1), // Unlimited reconnects
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Printf("Reconnected to NATS at %s", nc.ConnectedUrl())
		}),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			log.Printf("NATS connection lost: %v", err)
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Create JetStream context
	js, err := nc.JetStream(nats.PublishAsyncMaxPending(256))
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	service := &NATSStreamingService{
		nc:      nc,
		js:      js,
		config:  config,
		streams: make(map[string]*StreamInfo),
	}

	// Initialize streams
	if err := service.initializeStreams(); err != nil {
		service.Close()
		return nil, fmt.Errorf("failed to initialize streams: %w", err)
	}

	log.Println("NATS JetStream service initialized successfully")
	return service, nil
}

// initializeStreams creates the necessary streams
func (nss *NATSStreamingService) initializeStreams() error {
	// News articles stream
	newsStream := &nats.StreamConfig{
		Name:      "NEWS_ARTICLES",
		Subjects:  []string{"news.articles.*", "news.updates.*"},
		Retention: nats.LimitsPolicy,
		MaxAge:    24 * time.Hour,
		MaxBytes:  100 * 1024 * 1024, // 100MB
		MaxMsgs:   10000,
		Replicas:  1,
		Storage:   nats.FileStorage,
	}

	if err := nss.createStream(newsStream); err != nil {
		return err
	}

	// Analytics stream
	analyticsStream := &nats.StreamConfig{
		Name:      "NEWS_ANALYTICS",
		Subjects:  []string{"analytics.*", "metrics.*"},
		Retention: nats.LimitsPolicy,
		MaxAge:    7 * 24 * time.Hour, // 7 days
		MaxBytes:  500 * 1024 * 1024,  // 500MB
		MaxMsgs:   50000,
		Replicas:  1,
		Storage:   nats.FileStorage,
	}

	if err := nss.createStream(analyticsStream); err != nil {
		return err
	}

	// Real-time events stream
	eventsStream := &nats.StreamConfig{
		Name:      "NEWS_EVENTS",
		Subjects:  []string{"events.*", "alerts.*"},
		Retention: nats.LimitsPolicy,
		MaxAge:    1 * time.Hour,
		MaxBytes:  50 * 1024 * 1024, // 50MB
		MaxMsgs:   5000,
		Replicas:  1,
		Storage:   nats.MemoryStorage, // In-memory for speed
	}

	return nss.createStream(eventsStream)
}

// createStream creates or updates a stream
func (nss *NATSStreamingService) createStream(config *nats.StreamConfig) error {
	// Check if stream exists
	stream, err := nss.js.StreamInfo(config.Name)
	if err != nil {
		// Stream doesn't exist, create it
		stream, err = nss.js.AddStream(config)
		if err != nil {
			return fmt.Errorf("failed to create stream %s: %w", config.Name, err)
		}
		log.Printf("Created stream: %s", config.Name)
	} else {
		// Stream exists, update if needed
		log.Printf("Stream %s already exists with %d messages", config.Name, stream.State.Msgs)
	}

	// Store stream info
	nss.streams[config.Name] = &StreamInfo{
		Name:     config.Name,
		Subjects: config.Subjects,
	}

	return nil
}

// PublishArticle publishes an article event
func (nss *NATSStreamingService) PublishArticle(article model.Article, eventType string) error {
	event := NewsEvent{
		Type:      eventType, // "article_published" or "article_updated"
		Timestamp: time.Now(),
		Source:    "news-service",
		Region:    article.Topic,
		Data: EventData{
			Article: &article,
		},
	}

	subject := fmt.Sprintf("news.articles.%s", article.Topic)
	return nss.publishEvent(subject, event)
}

// PublishAnalytics publishes analytics data
func (nss *NATSStreamingService) PublishAnalytics(analytics AnalyticsData) error {
	event := NewsEvent{
		Type:      "analytics",
		Timestamp: time.Now(),
		Source:    "news-service",
		Data: EventData{
			Analytics: &analytics,
		},
	}

	subject := "analytics.engagement"
	return nss.publishEvent(subject, event)
}

// PublishTrending publishes trending topic data
func (nss *NATSStreamingService) PublishTrending(trending TrendingData, region string) error {
	event := NewsEvent{
		Type:      "trending_topic",
		Timestamp: time.Now(),
		Source:    "news-service",
		Region:    region,
		Data: EventData{
			Trending: &trending,
		},
	}

	subject := fmt.Sprintf("events.trending.%s", region)
	return nss.publishEvent(subject, event)
}

// PublishMetrics publishes system metrics
func (nss *NATSStreamingService) PublishMetrics(metrics MetricsData) error {
	event := NewsEvent{
		Type:      "metrics",
		Timestamp: time.Now(),
		Source:    "news-service",
		Data: EventData{
			Metrics: &metrics,
		},
	}

	subject := "metrics.system"
	return nss.publishEvent(subject, event)
}

// publishEvent is a helper to publish events
func (nss *NATSStreamingService) publishEvent(subject string, event NewsEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Publish with acknowledgment
	_, err = nss.js.Publish(subject, data)
	if err != nil {
		return fmt.Errorf("failed to publish to subject %s: %w", subject, err)
	}

	log.Printf("Published event: type=%s, subject=%s", event.Type, subject)
	return nil
}

// SubscribeToArticles subscribes to article events with a consumer
func (nss *NATSStreamingService) SubscribeToArticles(region string, handler func(NewsEvent) error) error {
	subject := fmt.Sprintf("news.articles.%s", region)
	consumerName := fmt.Sprintf("articles-consumer-%s", region)

	return nss.createDurableConsumer("NEWS_ARTICLES", consumerName, subject, handler)
}

// SubscribeToAnalytics subscribes to analytics events
func (nss *NATSStreamingService) SubscribeToAnalytics(handler func(NewsEvent) error) error {
	return nss.createDurableConsumer("NEWS_ANALYTICS", "analytics-consumer", "analytics.*", handler)
}

// SubscribeToEvents subscribes to real-time events
func (nss *NATSStreamingService) SubscribeToEvents(eventType string, handler func(NewsEvent) error) error {
	subject := fmt.Sprintf("events.%s.*", eventType)
	consumerName := fmt.Sprintf("events-consumer-%s", eventType)

	return nss.createDurableConsumer("NEWS_EVENTS", consumerName, subject, handler)
}

// createDurableConsumer creates a durable consumer for a stream
func (nss *NATSStreamingService) createDurableConsumer(streamName, consumerName, subject string, handler func(NewsEvent) error) error {
	// Consumer configuration
	consumerConfig := &nats.ConsumerConfig{
		Durable:       consumerName,
		FilterSubject: subject,
		AckPolicy:     nats.AckExplicitPolicy,
		MaxDeliver:    3,
		AckWait:       30 * time.Second,
		ReplayPolicy:  nats.ReplayInstantPolicy,
	}

	// Create or get consumer
	_, err := nss.js.AddConsumer(streamName, consumerConfig)
	if err != nil {
		return fmt.Errorf("failed to create consumer %s: %w", consumerName, err)
	}

	// Subscribe to the consumer
	_, err = nss.js.Subscribe(subject, func(msg *nats.Msg) {
		var event NewsEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			log.Printf("Failed to unmarshal event: %v", err)
			msg.Nak()
			return
		}

		// Process the event
		if err := handler(event); err != nil {
			log.Printf("Failed to handle event: %v", err)
			msg.Nak()
			return
		}

		// Acknowledge successful processing
		msg.Ack()
	}, nats.Durable(consumerName), nats.ManualAck())

	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", subject, err)
	}

	log.Printf("Created durable consumer: %s for stream: %s", consumerName, streamName)
	return nil
}

// GetStreamInfo returns information about a stream
func (nss *NATSStreamingService) GetStreamInfo(streamName string) (*nats.StreamInfo, error) {
	return nss.js.StreamInfo(streamName)
}

// GetConsumerInfo returns information about a consumer
func (nss *NATSStreamingService) GetConsumerInfo(streamName, consumerName string) (*nats.ConsumerInfo, error) {
	return nss.js.ConsumerInfo(streamName, consumerName)
}

// Close closes the NATS connection
func (nss *NATSStreamingService) Close() {
	if nss.nc != nil {
		nss.nc.Close()
		log.Println("NATS JetStream service closed")
	}
}

// Health check methods for monitoring
func (nss *NATSStreamingService) IsConnected() bool {
	return nss.nc != nil && nss.nc.IsConnected()
}

func (nss *NATSStreamingService) GetConnectionStats() map[string]interface{} {
	if nss.nc == nil {
		return map[string]interface{}{"connected": false}
	}

	stats := nss.nc.Stats()
	return map[string]interface{}{
		"connected":  nss.nc.IsConnected(),
		"url":        nss.nc.ConnectedUrl(),
		"in_msgs":    stats.InMsgs,
		"out_msgs":   stats.OutMsgs,
		"in_bytes":   stats.InBytes,
		"out_bytes":  stats.OutBytes,
		"reconnects": stats.Reconnects,
	}
}
