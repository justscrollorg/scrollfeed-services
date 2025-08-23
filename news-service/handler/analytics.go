package handler

import (
	"fmt"
	"log"
	"news-service/model"
	"sync"
	"time"
)

// AnalyticsProcessor processes news analytics in real-time
type AnalyticsProcessor struct {
	streaming     *NATSStreamingService
	metrics       *MetricsCollector
	trendingCache map[string]*TrendingCache
	mu            sync.RWMutex
}

// TrendingCache holds trending topic data
type TrendingCache struct {
	Topic      string
	Score      float64
	LastUpdate time.Time
	Articles   []string
	Keywords   []string
}

// MetricsCollector collects system metrics
type MetricsCollector struct {
	RequestCounts map[string]int64
	ErrorCounts   map[string]int64
	ResponseTimes map[string][]time.Duration
	mu            sync.RWMutex
}

// NewAnalyticsProcessor creates a new analytics processor
func NewAnalyticsProcessor(streaming *NATSStreamingService) *AnalyticsProcessor {
	processor := &AnalyticsProcessor{
		streaming:     streaming,
		metrics:       NewMetricsCollector(),
		trendingCache: make(map[string]*TrendingCache),
	}

	// Start analytics consumers
	go processor.startConsumers()

	// Start metrics publishing
	go processor.startMetricsPublisher()

	return processor
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		RequestCounts: make(map[string]int64),
		ErrorCounts:   make(map[string]int64),
		ResponseTimes: make(map[string][]time.Duration),
	}
}

// startConsumers starts all analytics consumers
func (ap *AnalyticsProcessor) startConsumers() {
	// Consumer for article events to generate analytics
	err := ap.streaming.SubscribeToArticles("*", ap.handleArticleEvent)
	if err != nil {
		log.Printf("Failed to subscribe to article events: %v", err)
	}

	// Consumer for analytics events
	err = ap.streaming.SubscribeToAnalytics(ap.handleAnalyticsEvent)
	if err != nil {
		log.Printf("Failed to subscribe to analytics events: %v", err)
	}

	// Consumer for trending events
	err = ap.streaming.SubscribeToEvents("trending", ap.handleTrendingEvent)
	if err != nil {
		log.Printf("Failed to subscribe to trending events: %v", err)
	}

	log.Println("Analytics consumers started")
}

// handleArticleEvent processes incoming article events
func (ap *AnalyticsProcessor) handleArticleEvent(event NewsEvent) error {
	if event.Data.Article == nil {
		return fmt.Errorf("no article data in event")
	}

	article := event.Data.Article
	log.Printf("Processing article event: %s in region: %s", article.Title, event.Region)

	// Generate analytics data
	analytics := AnalyticsData{
		ArticleID:      fmt.Sprintf("%s-%d", article.URL, time.Now().Unix()),
		ViewCount:      1, // Simulate initial view
		ShareCount:     0,
		EngagementRate: calculateEngagementRate(article),
		Demographics:   generateDemographics(event.Region),
		Tags:           extractTags(article),
	}

	// Publish analytics
	if err := ap.streaming.PublishAnalytics(analytics); err != nil {
		log.Printf("Failed to publish analytics: %v", err)
		return err
	}

	// Update trending topics
	ap.updateTrending(article, event.Region)

	return nil
}

// handleAnalyticsEvent processes analytics events
func (ap *AnalyticsProcessor) handleAnalyticsEvent(event NewsEvent) error {
	if event.Data.Analytics == nil {
		return fmt.Errorf("no analytics data in event")
	}

	analytics := event.Data.Analytics
	log.Printf("Processing analytics event for article: %s", analytics.ArticleID)

	// Store analytics in cache or database
	// This is where you would typically save to a time-series database like InfluxDB

	return nil
}

// handleTrendingEvent processes trending topic events
func (ap *AnalyticsProcessor) handleTrendingEvent(event NewsEvent) error {
	if event.Data.Trending == nil {
		return fmt.Errorf("no trending data in event")
	}

	trending := event.Data.Trending
	log.Printf("Processing trending event: %s in region: %s", trending.Topic, event.Region)

	// Update trending cache
	ap.mu.Lock()
	cacheKey := fmt.Sprintf("%s-%s", event.Region, trending.Topic)
	ap.trendingCache[cacheKey] = &TrendingCache{
		Topic:      trending.Topic,
		Score:      trending.Score,
		LastUpdate: event.Timestamp,
		Articles:   trending.Articles,
		Keywords:   trending.Keywords,
	}
	ap.mu.Unlock()

	return nil
}

// updateTrending updates trending topics based on new articles
func (ap *AnalyticsProcessor) updateTrending(article *model.Article, region string) {
	// Simple trending algorithm based on keywords in title
	keywords := extractTags(article)

	for _, keyword := range keywords {
		trending := TrendingData{
			Topic:     keyword,
			Score:     calculateTrendingScore(keyword, region),
			Articles:  []string{article.URL},
			Keywords:  []string{keyword},
			TrendType: "rising",
		}

		// Publish trending event
		if err := ap.streaming.PublishTrending(trending, region); err != nil {
			log.Printf("Failed to publish trending event: %v", err)
		}
	}
}

// startMetricsPublisher publishes system metrics periodically
func (ap *AnalyticsProcessor) startMetricsPublisher() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ap.publishSystemMetrics()
		}
	}
}

// publishSystemMetrics publishes current system metrics
func (ap *AnalyticsProcessor) publishSystemMetrics() {
	ap.metrics.mu.RLock()
	totalRequests := int64(0)
	totalErrors := int64(0)

	for _, count := range ap.metrics.RequestCounts {
		totalRequests += count
	}

	for _, count := range ap.metrics.ErrorCounts {
		totalErrors += count
	}

	errorRate := float64(0)
	if totalRequests > 0 {
		errorRate = float64(totalErrors) / float64(totalRequests)
	}
	ap.metrics.mu.RUnlock()

	metrics := MetricsData{
		ServiceName:  "news-service",
		RequestCount: totalRequests,
		ErrorRate:    errorRate,
		ResponseTime: ap.calculateAverageResponseTime(),
		CustomMetrics: map[string]interface{}{
			"trending_topics_count": len(ap.trendingCache),
			"uptime_seconds":        time.Since(time.Now().Add(-time.Hour)).Seconds(),
		},
	}

	if err := ap.streaming.PublishMetrics(metrics); err != nil {
		log.Printf("Failed to publish metrics: %v", err)
	}
}

// RecordRequest records a request for metrics
func (ap *AnalyticsProcessor) RecordRequest(endpoint string, duration time.Duration, success bool) {
	ap.metrics.mu.Lock()
	defer ap.metrics.mu.Unlock()

	ap.metrics.RequestCounts[endpoint]++

	if !success {
		ap.metrics.ErrorCounts[endpoint]++
	}

	if ap.metrics.ResponseTimes[endpoint] == nil {
		ap.metrics.ResponseTimes[endpoint] = make([]time.Duration, 0)
	}
	ap.metrics.ResponseTimes[endpoint] = append(ap.metrics.ResponseTimes[endpoint], duration)

	// Keep only last 100 response times
	if len(ap.metrics.ResponseTimes[endpoint]) > 100 {
		ap.metrics.ResponseTimes[endpoint] = ap.metrics.ResponseTimes[endpoint][1:]
	}
}

// GetTrendingTopics returns current trending topics
func (ap *AnalyticsProcessor) GetTrendingTopics(region string) []TrendingCache {
	ap.mu.RLock()
	defer ap.mu.RUnlock()

	var trending []TrendingCache
	for key, cache := range ap.trendingCache {
		if region == "" || fmt.Sprintf("%s-", region) == key[:len(region)+1] {
			trending = append(trending, *cache)
		}
	}

	return trending
}

// Helper functions

func calculateEngagementRate(article *model.Article) float64 {
	// Simple engagement calculation based on title length and content
	score := float64(len(article.Title)) / 100.0
	if len(article.Description) > 200 {
		score += 0.2
	}
	if article.Image != "" {
		score += 0.1
	}
	return score
}

func generateDemographics(region string) map[string]int64 {
	// Simulate demographics based on region
	demographics := make(map[string]int64)

	switch region {
	case "us":
		demographics["18-24"] = 15
		demographics["25-34"] = 30
		demographics["35-44"] = 25
		demographics["45-54"] = 20
		demographics["55+"] = 10
	case "in":
		demographics["18-24"] = 25
		demographics["25-34"] = 35
		demographics["35-44"] = 25
		demographics["45-54"] = 10
		demographics["55+"] = 5
	default:
		demographics["18-24"] = 20
		demographics["25-34"] = 30
		demographics["35-44"] = 25
		demographics["45-54"] = 15
		demographics["55+"] = 10
	}

	return demographics
}

func extractTags(article *model.Article) []string {
	// Simple keyword extraction from title
	// In production, you'd use NLP libraries
	keywords := []string{}

	// Common news keywords
	commonWords := map[string]bool{
		"the": true, "and": true, "or": true, "but": true, "in": true,
		"on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "a": true, "an": true, "is": true,
		"are": true, "was": true, "were": true, "be": true, "been": true,
	}

	words := []string{"technology", "politics", "business", "sports", "health"}
	for _, word := range words {
		if len(word) > 3 && !commonWords[word] {
			keywords = append(keywords, word)
		}
	}

	// Limit to top 5 keywords
	if len(keywords) > 5 {
		keywords = keywords[:5]
	}

	return keywords
}

func calculateTrendingScore(keyword, region string) float64 {
	// Simple trending score calculation
	// In production, this would be much more sophisticated
	base := 1.0

	// Boost based on region
	if region == "us" {
		base *= 1.2
	} else if region == "in" {
		base *= 1.1
	}

	// Add some randomness to simulate real trending
	return base + (float64(time.Now().Unix()%10) / 10.0)
}

func (ap *AnalyticsProcessor) calculateAverageResponseTime() time.Duration {
	ap.metrics.mu.RLock()
	defer ap.metrics.mu.RUnlock()

	total := time.Duration(0)
	count := 0

	for _, times := range ap.metrics.ResponseTimes {
		for _, t := range times {
			total += t
			count++
		}
	}

	if count == 0 {
		return 0
	}

	return total / time.Duration(count)
}
