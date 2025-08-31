package api

import (
	"log"
	"news-service/handler"
	"news-service/metrics"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.mongodb.org/mongo-driver/mongo"
)

var dbmngo *mongo.Database
var natsNewsHandler *handler.NewsHandler
var streamingAPI *StreamingAPI

func StartServer(db *mongo.Database, natsURL string) {
	router := gin.Default()

	// Add metrics middleware
	router.Use(metricsMiddleware())

	dbmngo = db

	// Initialize News Handler with the collection
	natsNewsHandler = handler.NewNewsHandler(db.Collection("articles"))

	// Initialize Streaming API
	streamingAPI = NewStreamingAPI(natsNewsHandler)

	// Health check routes
	router.GET("/", healthCheck)
	router.GET("/health", healthCheck)
	router.GET("/ready", healthCheck)
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API routes
	router.GET("/news-api/news", callnewsHandler)
	router.GET("/news-api/regions", getRegions)
	router.GET("/news-api/stats", getStats)
	router.POST("/news-api/fetch/:region", triggerRegionFetch)
	router.POST("/news-api/fetch-all", triggerAllFetch)

	// Streaming API routes
	streamingRoutes := router.Group("/streaming-api")
	{
		streamingRoutes.GET("/status", streamingAPI.GetStreamStatus)
		streamingRoutes.GET("/trending", streamingAPI.GetTrendingTopics)
		streamingRoutes.GET("/metrics", streamingAPI.GetAnalyticsMetrics)
		streamingRoutes.POST("/test-event", streamingAPI.PublishTestEvent)
		streamingRoutes.GET("/stream/:stream/metrics", streamingAPI.GetStreamMetrics)
		streamingRoutes.POST("/simulate-load", streamingAPI.SimulateLoad)
	}

	log.Println("News API is running at :80")
	log.Println("Streaming API available at /streaming-api/*")

	// Start the background fetcher
	go handler.StartScheduledFetcher(db)

	router.Run(":80")
}

func healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{"status": "healthy", "service": "news-service"})
}

func callnewsHandler(c *gin.Context) {
	log.Printf("[INFO] callnewsHandler called - using enhanced handler")
	enhancedNewsHandler(c, dbmngo)
}

func triggerRegionFetch(c *gin.Context) {
	region := c.Param("region")
	priority := c.DefaultQuery("priority", "normal")

	log.Printf("[INFO] Manual fetch triggered for region=%s, priority=%s", region, priority)

	if err := natsNewsHandler.TriggerNewsFetch(region, priority); err != nil {
		log.Printf("Failed to trigger fetch for region %s: %v", region, err)
		c.JSON(500, gin.H{"error": "Failed to trigger fetch", "details": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "Fetch triggered successfully", "region": region, "priority": priority})
}

func triggerAllFetch(c *gin.Context) {
	priority := c.DefaultQuery("priority", "high")

	log.Printf("[INFO] Manual fetch triggered for all regions, priority=%s", priority)

	if err := natsNewsHandler.TriggerAllRegionsFetch(priority); err != nil {
		log.Printf("Failed to trigger fetch for all regions: %v", err)
		c.JSON(500, gin.H{"error": "Failed to trigger fetch", "details": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "Fetch triggered for all regions", "priority": priority})
}

func getRegions(c *gin.Context) {
	// Return available regions
	regions := []string{"us", "in", "de"}
	c.JSON(200, gin.H{"regions": regions})
}

func getStats(c *gin.Context) {
	statsHandler(c, dbmngo)
}

// Metrics middleware
func metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		
		c.Next()
		
		duration := time.Since(start)
		status := strconv.Itoa(c.Writer.Status())
		
		metrics.HttpRequestsTotal.WithLabelValues(
			c.Request.Method,
			c.Request.URL.Path,
			status,
			"news-service",
		).Inc()
		
		metrics.HttpRequestDuration.WithLabelValues(
			c.Request.Method,
			c.Request.URL.Path,
			"news-service",
		).Observe(duration.Seconds())
	}
}
