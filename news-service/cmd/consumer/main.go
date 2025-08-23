package main

import (
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
)

// EventData represents the events we'll consume
type NewsEvent struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Source    string    `json:"source"`
	Region    string    `json:"region"`
	Data      EventData `json:"data"`
}

type EventData struct {
	Article   *Article      `json:"article,omitempty"`
	Analytics *Analytics    `json:"analytics,omitempty"`
	Trending  *TrendingData `json:"trending,omitempty"`
	Metrics   *MetricsData  `json:"metrics,omitempty"`
}

type Article struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	URL         string `json:"url"`
	Image       string `json:"image"`
	Source      struct {
		Name string `json:"name"`
	} `json:"source"`
	PublishedAt time.Time `json:"publishedAt"`
	Topic       string    `json:"topic"`
	FetchedAt   time.Time `json:"fetchedAt"`
}

type Analytics struct {
	ArticleID      string           `json:"article_id"`
	ViewCount      int64            `json:"view_count"`
	ShareCount     int64            `json:"share_count"`
	EngagementRate float64          `json:"engagement_rate"`
	Demographics   map[string]int64 `json:"demographics"`
	Tags           []string         `json:"tags"`
}

type TrendingData struct {
	Topic     string   `json:"topic"`
	Score     float64  `json:"score"`
	Articles  []string `json:"articles"`
	Keywords  []string `json:"keywords"`
	TrendType string   `json:"trend_type"`
}

type MetricsData struct {
	ServiceName   string                 `json:"service_name"`
	RequestCount  int64                  `json:"request_count"`
	ErrorRate     float64                `json:"error_rate"`
	ResponseTime  time.Duration          `json:"response_time"`
	CustomMetrics map[string]interface{} `json:"custom_metrics"`
}

func main() {
	// Get NATS URL from environment
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	log.Printf("Connecting to NATS at %s", natsURL)

	// Connect to NATS
	nc, err := nats.Connect(natsURL,
		nats.Name("news-consumer"),
		nats.ReconnectWait(2*time.Second),
		nats.MaxReconnects(-1),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Printf("Reconnected to NATS at %s", nc.ConnectedUrl())
		}),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			log.Printf("NATS connection lost: %v", err)
		}),
	)
	if err != nil {
		log.Fatal("Failed to connect to NATS:", err)
	}
	defer nc.Close()

	// Create JetStream context
	js, err := nc.JetStream()
	if err != nil {
		log.Fatal("Failed to create JetStream context:", err)
	}

	log.Println("Connected to NATS JetStream successfully")

	// Set up signal handling for graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Start consumers
	startConsumers(js)

	// Wait for interrupt signal
	<-c
	log.Println("Shutting down news event consumer...")
}

func startConsumers(js nats.JetStreamContext) {
	// Consumer for news articles
	go consumeArticles(js)

	// Consumer for analytics events
	go consumeAnalytics(js)

	// Consumer for trending topics
	go consumeTrending(js)

	// Consumer for metrics
	go consumeMetrics(js)

	log.Println("All consumers started")
}

// consumeArticles consumes news article events
func consumeArticles(js nats.JetStreamContext) {
	subject := "news.articles.*"
	consumerName := "articles-processor"

	// Create durable consumer
	_, err := js.AddConsumer("NEWS_ARTICLES", &nats.ConsumerConfig{
		Durable:       consumerName,
		FilterSubject: subject,
		AckPolicy:     nats.AckExplicitPolicy,
		MaxDeliver:    3,
		AckWait:       30 * time.Second,
		ReplayPolicy:  nats.ReplayInstantPolicy,
	})
	if err != nil {
		log.Printf("Failed to create articles consumer: %v", err)
		return
	}

	// Subscribe to the consumer
	_, err = js.Subscribe(subject, func(msg *nats.Msg) {
		var event NewsEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			log.Printf("Failed to unmarshal article event: %v", err)
			msg.Nak()
			return
		}

		if event.Data.Article != nil {
			processArticle(event)
		}

		msg.Ack()
	}, nats.Durable(consumerName), nats.ManualAck())

	if err != nil {
		log.Printf("Failed to subscribe to articles: %v", err)
		return
	}

	log.Printf("Articles consumer started for subject: %s", subject)
}

// consumeAnalytics consumes analytics events
func consumeAnalytics(js nats.JetStreamContext) {
	subject := "analytics.*"
	consumerName := "analytics-processor"

	_, err := js.AddConsumer("NEWS_ANALYTICS", &nats.ConsumerConfig{
		Durable:       consumerName,
		FilterSubject: subject,
		AckPolicy:     nats.AckExplicitPolicy,
		MaxDeliver:    3,
		AckWait:       30 * time.Second,
		ReplayPolicy:  nats.ReplayInstantPolicy,
	})
	if err != nil {
		log.Printf("Failed to create analytics consumer: %v", err)
		return
	}

	_, err = js.Subscribe(subject, func(msg *nats.Msg) {
		var event NewsEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			log.Printf("Failed to unmarshal analytics event: %v", err)
			msg.Nak()
			return
		}

		if event.Data.Analytics != nil {
			processAnalytics(event)
		}

		msg.Ack()
	}, nats.Durable(consumerName), nats.ManualAck())

	if err != nil {
		log.Printf("Failed to subscribe to analytics: %v", err)
		return
	}

	log.Printf("Analytics consumer started for subject: %s", subject)
}

// consumeTrending consumes trending topic events
func consumeTrending(js nats.JetStreamContext) {
	subject := "events.trending.*"
	consumerName := "trending-processor"

	_, err := js.AddConsumer("NEWS_EVENTS", &nats.ConsumerConfig{
		Durable:       consumerName,
		FilterSubject: subject,
		AckPolicy:     nats.AckExplicitPolicy,
		MaxDeliver:    3,
		AckWait:       30 * time.Second,
		ReplayPolicy:  nats.ReplayInstantPolicy,
	})
	if err != nil {
		log.Printf("Failed to create trending consumer: %v", err)
		return
	}

	_, err = js.Subscribe(subject, func(msg *nats.Msg) {
		var event NewsEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			log.Printf("Failed to unmarshal trending event: %v", err)
			msg.Nak()
			return
		}

		if event.Data.Trending != nil {
			processTrending(event)
		}

		msg.Ack()
	}, nats.Durable(consumerName), nats.ManualAck())

	if err != nil {
		log.Printf("Failed to subscribe to trending: %v", err)
		return
	}

	log.Printf("Trending consumer started for subject: %s", subject)
}

// consumeMetrics consumes system metrics events
func consumeMetrics(js nats.JetStreamContext) {
	subject := "metrics.*"
	consumerName := "metrics-processor"

	_, err := js.AddConsumer("NEWS_ANALYTICS", &nats.ConsumerConfig{
		Durable:       consumerName + "_metrics",
		FilterSubject: subject,
		AckPolicy:     nats.AckExplicitPolicy,
		MaxDeliver:    3,
		AckWait:       30 * time.Second,
		ReplayPolicy:  nats.ReplayInstantPolicy,
	})
	if err != nil {
		log.Printf("Failed to create metrics consumer: %v", err)
		return
	}

	_, err = js.Subscribe(subject, func(msg *nats.Msg) {
		var event NewsEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			log.Printf("Failed to unmarshal metrics event: %v", err)
			msg.Nak()
			return
		}

		if event.Data.Metrics != nil {
			processMetrics(event)
		}

		msg.Ack()
	}, nats.Durable(consumerName+"_metrics"), nats.ManualAck())

	if err != nil {
		log.Printf("Failed to subscribe to metrics: %v", err)
		return
	}

	log.Printf("Metrics consumer started for subject: %s", subject)
}

// Event processing functions - this is where you'd implement your business logic

func processArticle(event NewsEvent) {
	article := event.Data.Article
	log.Printf("📰 [ARTICLE] Region: %s | Title: %s | Source: %s | Published: %s",
		event.Region,
		truncateString(article.Title, 50),
		article.Source.Name,
		article.PublishedAt.Format("15:04:05"),
	)

	// Here you would:
	// - Index the article for search
	// - Extract entities and keywords
	// - Calculate sentiment
	// - Update recommendation systems
	// - Trigger notifications for subscribers
}

func processAnalytics(event NewsEvent) {
	analytics := event.Data.Analytics
	log.Printf("📊 [ANALYTICS] Article: %s | Views: %d | Shares: %d | Engagement: %.2f",
		truncateString(analytics.ArticleID, 20),
		analytics.ViewCount,
		analytics.ShareCount,
		analytics.EngagementRate,
	)

	// Here you would:
	// - Store analytics in time-series database
	// - Update dashboards
	// - Trigger alerts for unusual patterns
	// - Calculate trending scores
}

func processTrending(event NewsEvent) {
	trending := event.Data.Trending
	log.Printf("🔥 [TRENDING] Region: %s | Topic: %s | Score: %.1f | Type: %s",
		event.Region,
		trending.Topic,
		trending.Score,
		trending.TrendType,
	)

	// Here you would:
	// - Update trending topics cache
	// - Send push notifications
	// - Update recommendation algorithms
	// - Trigger content creation workflows
}

func processMetrics(event NewsEvent) {
	metrics := event.Data.Metrics
	log.Printf("⚡ [METRICS] Service: %s | Requests: %d | Error Rate: %.3f | Response Time: %v",
		metrics.ServiceName,
		metrics.RequestCount,
		metrics.ErrorRate,
		metrics.ResponseTime,
	)

	// Here you would:
	// - Store metrics in monitoring system
	// - Update alerting rules
	// - Calculate SLIs/SLOs
	// - Trigger auto-scaling if needed
}

// Utility function to truncate strings for logging
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
