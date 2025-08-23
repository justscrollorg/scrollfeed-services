package handler

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"news-service/model"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type NewsHandler struct {
	collection         *mongo.Collection
	config             *NewsConfig
	strategies         map[string]NewsStrategy
	natsPublisher      *NATSPublisher
	streamingService   *NATSStreamingService
	analyticsProcessor *AnalyticsProcessor
}

// Configuration struct for news fetching
type NewsConfig struct {
	APIKey           string
	BaseURL          string
	Regions          []string
	MaxPages         int
	MaxArticles      int
	RateLimit        time.Duration
	FetchInterval    time.Duration
	EnableNATS       bool
	NATSConfig       *NATSConfig
	RegionStrategies map[string]string
	EnableJetStream  bool
	StreamingConfig  *StreamingConfig
}

func NewNewsHandler(collection *mongo.Collection) *NewsHandler {
	config := loadImprovedNewsConfig()

	// Initialize NATS if enabled
	var natsPublisher *NATSPublisher
	if config.EnableNATS {
		var err error
		natsPublisher, err = NewNATSPublisher(config.NATSConfig)
		if err != nil {
			log.Printf("Failed to initialize NATS publisher: %v", err)
		} else {
			log.Println("NATS publisher initialized successfully")
		}
	}

	// Initialize JetStream if enabled
	var streamingService *NATSStreamingService
	var analyticsProcessor *AnalyticsProcessor
	if config.EnableJetStream {
		var err error
		streamingService, err = NewNATSStreamingService(config.StreamingConfig)
		if err != nil {
			log.Printf("Failed to initialize NATS JetStream: %v", err)
		} else {
			log.Println("NATS JetStream initialized successfully")
			// Initialize analytics processor
			analyticsProcessor = NewAnalyticsProcessor(streamingService)
		}
	}

	// Initialize strategies
	strategies := make(map[string]NewsStrategy)
	strategies["api"] = &APIStrategy{}
	strategies["rss"] = NewRSSStrategy()

	return &NewsHandler{
		collection:         collection,
		config:             config,
		strategies:         strategies,
		natsPublisher:      natsPublisher,
		streamingService:   streamingService,
		analyticsProcessor: analyticsProcessor,
	}
}

// Load configuration from environment variables
func loadImprovedNewsConfig() *NewsConfig {
	enableNATS, _ := strconv.ParseBool(getEnvOrDefault("ENABLE_NATS", "false"))
	enableJetStream, _ := strconv.ParseBool(getEnvOrDefault("ENABLE_JETSTREAM", "true"))

	var natsConfig *NATSConfig
	if enableNATS {
		natsConfig = &NATSConfig{
			URL:     getEnvOrDefault("NATS_URL", "nats://localhost:4222"),
			Subject: getEnvOrDefault("NATS_SUBJECT", "news.articles"),
		}
	}

	var streamingConfig *StreamingConfig
	if enableJetStream {
		streamingConfig = &StreamingConfig{
			URL:        getEnvOrDefault("NATS_URL", "nats://localhost:4222"),
			StreamName: "NEWS_STREAM",
			Subjects:   []string{"news.*", "analytics.*", "events.*"},
			MaxAge:     24 * time.Hour,
			MaxBytes:   100 * 1024 * 1024, // 100MB
			MaxMsgs:    10000,
			Replicas:   1,
		}
	}

	// Region-specific strategies: "in" uses RSS, others use API
	regionStrategies := make(map[string]string)
	regions := strings.Split(getEnvOrDefault("NEWS_REGIONS", "us,in,de"), ",")
	for _, region := range regions {
		if region == "in" {
			regionStrategies[region] = "rss"
		} else {
			regionStrategies[region] = "api"
		}
	}

	config := &NewsConfig{
		APIKey:           os.Getenv("NEWS_API_KEY"),
		BaseURL:          getEnvOrDefault("NEWS_API_BASE_URL", "https://newsapi.org/v2/top-headlines"),
		Regions:          regions,
		MaxPages:         getEnvIntOrDefault("NEWS_MAX_PAGES", 2),
		MaxArticles:      getEnvIntOrDefault("NEWS_MAX_ARTICLES", 50),
		RateLimit:        time.Duration(getEnvIntOrDefault("NEWS_RATE_LIMIT_SECONDS", 2)) * time.Second,
		FetchInterval:    time.Duration(getEnvIntOrDefault("NEWS_FETCH_INTERVAL_HOURS", 2)) * time.Hour,
		EnableNATS:       enableNATS,
		NATSConfig:       natsConfig,
		RegionStrategies: regionStrategies,
		EnableJetStream:  enableJetStream,
		StreamingConfig:  streamingConfig,
	}

	if config.APIKey == "" {
		log.Println("Warning: Missing NEWS_API_KEY environment variable - API strategy will not work")
	}

	log.Printf("Hybrid News Config: BaseURL=%s, Regions=%v, MaxPages=%d, MaxArticles=%d, NATS=%t, JetStream=%t",
		config.BaseURL, config.Regions, config.MaxPages, config.MaxArticles, config.EnableNATS, config.EnableJetStream)

	return config
}

// GetNews handles GET /news endpoint with strategy selection
func (nh *NewsHandler) GetNews(c *gin.Context) {
	region := c.Query("region")
	if region == "" {
		region = "us" // default
	}

	log.Printf("GetNews request for region: %s", region)

	// Get articles from database
	filter := bson.M{"topic": region}

	opts := options.Find().SetSort(bson.D{{Key: "fetchedat", Value: -1}}).SetLimit(50)
	cursor, err := nh.collection.Find(context.TODO(), filter, opts)
	if err != nil {
		log.Printf("Database query error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer cursor.Close(context.TODO())

	var articles []model.Article
	if err = cursor.All(context.TODO(), &articles); err != nil {
		log.Printf("Cursor decode error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Data parsing error"})
		return
	}

	log.Printf("Retrieved %d articles for region: %s", len(articles), region)
	c.JSON(http.StatusOK, articles)
}

// FetchNews manually triggers news fetching for a specific region
func (nh *NewsHandler) FetchNews(c *gin.Context) {
	region := c.Query("region")
	if region == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "region parameter required"})
		return
	}

	strategy := nh.config.RegionStrategies[region]
	if strategy == "" {
		// Force RSS for India, API for others
		if region == "in" {
			strategy = "rss"
		} else {
			strategy = "api" // fallback
		}
	}

	log.Printf("Manual fetch request for region: %s using strategy: %s", region, strategy)

	// Use the appropriate strategy
	newsStrategy, exists := nh.strategies[strategy]
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Strategy not available"})
		return
	}

	articles, err := newsStrategy.FetchNews(region, nh.config)
	if err != nil {
		log.Printf("Failed to fetch news for region %s: %v", region, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Fetch failed: %v", err)})
		return
	}

	// Store articles
	stored := nh.storeArticles(articles)

	// Publish to NATS if enabled
	if nh.natsPublisher != nil {
		if err := nh.natsPublisher.PublishBatch(articles); err != nil {
			log.Printf("Failed to publish to NATS: %v", err)
		}
	}

	// Publish to JetStream if enabled
	if nh.streamingService != nil {
		for _, article := range articles {
			if err := nh.streamingService.PublishArticle(article, "article_published"); err != nil {
				log.Printf("Failed to publish article to JetStream: %v", err)
			}
		}
	}

	// Record metrics
	if nh.analyticsProcessor != nil {
		nh.analyticsProcessor.RecordRequest(fmt.Sprintf("fetch_%s", region), time.Since(time.Now()), true)
	}

	c.JSON(http.StatusOK, gin.H{
		"region":              region,
		"strategy":            strategy,
		"fetched":             len(articles),
		"stored":              stored,
		"nats_published":      nh.natsPublisher != nil,
		"jetstream_published": nh.streamingService != nil,
	})
}

func (nh *NewsHandler) storeArticles(articles []model.Article) int {
	stored := 0
	for _, article := range articles {
		filter := bson.M{"url": article.URL}
		update := bson.M{"$set": article}

		result, err := nh.collection.UpdateOne(
			context.TODO(),
			filter,
			update,
			options.Update().SetUpsert(true),
		)

		if err != nil {
			log.Printf("Insert failed for article: %s | error: %v", article.URL, err)
		} else {
			if result.UpsertedCount > 0 || result.ModifiedCount > 0 {
				stored++
			}
		}
	}
	return stored
}

// TriggerNewsFetch manually triggers news fetching for a specific region
func (nh *NewsHandler) TriggerNewsFetch(region, priority string) error {
	strategy := nh.config.RegionStrategies[region]
	if strategy == "" {
		strategy = "api"
	}

	log.Printf("Triggering fetch for region=%s, strategy=%s, priority=%s", region, strategy, priority)

	newsStrategy, exists := nh.strategies[strategy]
	if !exists {
		return fmt.Errorf("strategy %s not available", strategy)
	}

	articles, err := newsStrategy.FetchNews(region, nh.config)
	if err != nil {
		return fmt.Errorf("fetch failed: %v", err)
	}

	stored := nh.storeArticles(articles)
	log.Printf("Manual fetch completed for region=%s: fetched=%d, stored=%d", region, len(articles), stored)

	// Publish to NATS if enabled
	if nh.natsPublisher != nil && len(articles) > 0 {
		if err := nh.natsPublisher.PublishBatch(articles); err != nil {
			log.Printf("Failed to publish to NATS: %v", err)
		}
	}

	// Publish to JetStream if enabled
	if nh.streamingService != nil && len(articles) > 0 {
		for _, article := range articles {
			if err := nh.streamingService.PublishArticle(article, "article_published"); err != nil {
				log.Printf("Failed to publish article to JetStream: %v", err)
			}
		}
	}

	return nil
}

// TriggerAllRegionsFetch triggers fetching for all configured regions
func (nh *NewsHandler) TriggerAllRegionsFetch(priority string) error {
	log.Printf("Triggering fetch for all regions with priority=%s", priority)

	for _, region := range nh.config.Regions {
		if err := nh.TriggerNewsFetch(region, priority); err != nil {
			log.Printf("Failed to fetch region %s: %v", region, err)
			continue
		}
		// Small delay between regions
		time.Sleep(1 * time.Second)
	}

	return nil
}

// StartScheduledFetcher runs the hybrid news fetcher
func StartScheduledFetcher(db *mongo.Database) {
	handler := NewNewsHandler(db.Collection("articles"))
	config := handler.config

	log.Println("Starting hybrid scheduled news fetcher...")

	// Run immediately
	handler.fetchAndStoreAllRegions()

	ticker := time.NewTicker(config.FetchInterval)
	for {
		<-ticker.C
		handler.fetchAndStoreAllRegions()
	}
}

func (nh *NewsHandler) fetchAndStoreAllRegions() {
	log.Println("Starting hybrid fetch cycle for all regions...")

	for _, region := range nh.config.Regions {
		strategy := nh.config.RegionStrategies[region]
		if strategy == "" {
			strategy = "api"
		}

		log.Printf("Fetching region=%s using strategy=%s", region, strategy)

		newsStrategy, exists := nh.strategies[strategy]
		if !exists {
			log.Printf("Strategy %s not available for region %s", strategy, region)
			continue
		}

		articles, err := newsStrategy.FetchNews(region, nh.config)
		if err != nil {
			log.Printf("Failed to fetch news for region=%s: %v", region, err)
			continue
		}

		stored := nh.storeArticles(articles)
		log.Printf("Region=%s: fetched=%d, stored=%d", region, len(articles), stored)

		// Publish to NATS if enabled
		if nh.natsPublisher != nil && len(articles) > 0 {
			if err := nh.natsPublisher.PublishBatch(articles); err != nil {
				log.Printf("Failed to publish to NATS for region %s: %v", region, err)
			}
		}

		// Publish to JetStream if enabled
		if nh.streamingService != nil && len(articles) > 0 {
			for _, article := range articles {
				if err := nh.streamingService.PublishArticle(article, "article_published"); err != nil {
					log.Printf("Failed to publish article to JetStream for region %s: %v", region, err)
				}
			}
		}

		// Rate limiting between regions
		time.Sleep(nh.config.RateLimit)
	}

	log.Println("Finished hybrid fetch cycle")
}

// Utility functions
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// Getter methods for accessing services
func (nh *NewsHandler) GetStreamingService() *NATSStreamingService {
	return nh.streamingService
}

func (nh *NewsHandler) GetAnalyticsProcessor() *AnalyticsProcessor {
	return nh.analyticsProcessor
}

func (nh *NewsHandler) GetConfig() *NewsConfig {
	return nh.config
}
