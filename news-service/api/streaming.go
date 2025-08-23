package api

import (
	"net/http"
	"news-service/handler"
	"strconv"

	"github.com/gin-gonic/gin"
)

// StreamingAPI handles streaming-related endpoints
type StreamingAPI struct {
	newsHandler *handler.NewsHandler
}

// NewStreamingAPI creates a new streaming API
func NewStreamingAPI(newsHandler *handler.NewsHandler) *StreamingAPI {
	return &StreamingAPI{newsHandler: newsHandler}
}

// GetStreamStatus returns the status of streaming services
func (sa *StreamingAPI) GetStreamStatus(c *gin.Context) {
	status := map[string]interface{}{
		"timestamp": "2025-08-22T00:00:00Z",
		"services": map[string]interface{}{
			"jetstream": map[string]interface{}{
				"enabled":   sa.newsHandler != nil,
				"connected": false,
				"streams":   []string{},
			},
			"analytics": map[string]interface{}{
				"enabled": sa.newsHandler != nil,
				"active":  true,
			},
		},
	}

	// Get actual status if services are available
	if sa.newsHandler != nil && sa.newsHandler.GetStreamingService() != nil {
		streaming := sa.newsHandler.GetStreamingService()
		status["services"].(map[string]interface{})["jetstream"].(map[string]interface{})["connected"] = streaming.IsConnected()
		status["services"].(map[string]interface{})["jetstream"].(map[string]interface{})["connection_stats"] = streaming.GetConnectionStats()

		// Get stream information
		if streamInfo, err := streaming.GetStreamInfo("NEWS_ARTICLES"); err == nil {
			status["services"].(map[string]interface{})["jetstream"].(map[string]interface{})["news_stream"] = map[string]interface{}{
				"name":     streamInfo.Config.Name,
				"subjects": streamInfo.Config.Subjects,
				"messages": streamInfo.State.Msgs,
				"bytes":    streamInfo.State.Bytes,
			}
		}

		if streamInfo, err := streaming.GetStreamInfo("NEWS_ANALYTICS"); err == nil {
			status["services"].(map[string]interface{})["jetstream"].(map[string]interface{})["analytics_stream"] = map[string]interface{}{
				"name":     streamInfo.Config.Name,
				"messages": streamInfo.State.Msgs,
				"bytes":    streamInfo.State.Bytes,
			}
		}
	}

	c.JSON(http.StatusOK, status)
}

// GetTrendingTopics returns trending topics from analytics
func (sa *StreamingAPI) GetTrendingTopics(c *gin.Context) {
	region := c.Query("region")

	if sa.newsHandler == nil || sa.newsHandler.GetAnalyticsProcessor() == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Analytics service not available",
		})
		return
	}

	trending := sa.newsHandler.GetAnalyticsProcessor().GetTrendingTopics(region)

	c.JSON(http.StatusOK, gin.H{
		"region":   region,
		"trending": trending,
		"count":    len(trending),
	})
}

// GetAnalyticsMetrics returns current analytics metrics
func (sa *StreamingAPI) GetAnalyticsMetrics(c *gin.Context) {
	if sa.newsHandler == nil || sa.newsHandler.GetAnalyticsProcessor() == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Analytics service not available",
		})
		return
	}

	// Return mock metrics for now - in production this would come from the analytics processor
	metrics := map[string]interface{}{
		"total_articles_processed": 1250,
		"articles_last_hour":       42,
		"top_regions": []map[string]interface{}{
			{"region": "us", "count": 450},
			{"region": "in", "count": 380},
			{"region": "de", "count": 220},
		},
		"engagement_rates": map[string]float64{
			"us": 0.65,
			"in": 0.72,
			"de": 0.58,
		},
		"trending_keywords": []string{"technology", "politics", "climate", "sports", "business"},
	}

	c.JSON(http.StatusOK, metrics)
}

// PublishTestEvent publishes a test event to demonstrate streaming
func (sa *StreamingAPI) PublishTestEvent(c *gin.Context) {
	eventType := c.DefaultQuery("type", "test")
	region := c.DefaultQuery("region", "us")

	if sa.newsHandler == nil || sa.newsHandler.GetStreamingService() == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Streaming service not available",
		})
		return
	}

	streaming := sa.newsHandler.GetStreamingService()

	switch eventType {
	case "trending":
		trending := handler.TrendingData{
			Topic:     "test-topic",
			Score:     85.5,
			Articles:  []string{"test-article-1", "test-article-2"},
			Keywords:  []string{"test", "demo", "streaming"},
			TrendType: "rising",
		}

		if err := streaming.PublishTrending(trending, region); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

	case "analytics":
		analytics := handler.AnalyticsData{
			ArticleID:      "test-article-123",
			ViewCount:      100,
			ShareCount:     15,
			EngagementRate: 0.85,
			Demographics: map[string]int64{
				"18-24": 25,
				"25-34": 35,
				"35-44": 25,
				"45-54": 15,
			},
			Tags: []string{"technology", "innovation"},
		}

		if err := streaming.PublishAnalytics(analytics); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

	case "metrics":
		metrics := handler.MetricsData{
			ServiceName:  "news-service",
			RequestCount: 1000,
			ErrorRate:    0.02,
			ResponseTime: 150000000, // 150ms in nanoseconds
			CustomMetrics: map[string]interface{}{
				"test_mode": true,
				"timestamp": "2025-08-22T00:00:00Z",
			},
		}

		if err := streaming.PublishMetrics(metrics); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid event type. Supported: trending, analytics, metrics",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Test event published successfully",
		"event_type": eventType,
		"region":     region,
	})
}

// GetStreamMetrics returns detailed stream metrics
func (sa *StreamingAPI) GetStreamMetrics(c *gin.Context) {
	streamName := c.Param("stream")

	if sa.newsHandler == nil || sa.newsHandler.GetStreamingService() == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Streaming service not available",
		})
		return
	}

	streaming := sa.newsHandler.GetStreamingService()
	streamInfo, err := streaming.GetStreamInfo(streamName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":  "Stream not found",
			"stream": streamName,
		})
		return
	}

	metrics := map[string]interface{}{
		"name":          streamInfo.Config.Name,
		"subjects":      streamInfo.Config.Subjects,
		"max_age":       streamInfo.Config.MaxAge.String(),
		"max_bytes":     streamInfo.Config.MaxBytes,
		"max_msgs":      streamInfo.Config.MaxMsgs,
		"current_msgs":  streamInfo.State.Msgs,
		"current_bytes": streamInfo.State.Bytes,
		"first_seq":     streamInfo.State.FirstSeq,
		"last_seq":      streamInfo.State.LastSeq,
		"created":       streamInfo.Created,
	}

	c.JSON(http.StatusOK, metrics)
}

// SimulateLoad simulates load for testing streaming performance
func (sa *StreamingAPI) SimulateLoad(c *gin.Context) {
	countStr := c.DefaultQuery("count", "10")
	count, err := strconv.Atoi(countStr)
	if err != nil || count < 1 || count > 100 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid count parameter (1-100)",
		})
		return
	}

	if sa.newsHandler == nil || sa.newsHandler.GetStreamingService() == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Streaming service not available",
		})
		return
	}

	streaming := sa.newsHandler.GetStreamingService()
	published := 0

	for i := 0; i < count; i++ {
		analytics := handler.AnalyticsData{
			ArticleID:      "load-test-" + strconv.Itoa(i),
			ViewCount:      int64(i * 10),
			ShareCount:     int64(i * 2),
			EngagementRate: float64(i) / float64(count),
			Demographics: map[string]int64{
				"test": int64(i),
			},
			Tags: []string{"load", "test"},
		}

		if err := streaming.PublishAnalytics(analytics); err != nil {
			continue
		}
		published++
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Load simulation completed",
		"requested": count,
		"published": published,
	})
}
