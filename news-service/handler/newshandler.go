package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"news-service/model"
	"os"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Configuration struct for news fetching
type NewsConfig struct {
	APIKey        string
	BaseURL       string
	Regions       []string
	MaxPages      int
	MaxArticles   int
	RateLimit     time.Duration
	FetchInterval time.Duration
}

// Load configuration from environment variables
func loadNewsConfig() *NewsConfig {
	config := &NewsConfig{
		APIKey:        os.Getenv("NEWS_API_KEY"), // Generic name instead of GNEWS_API_KEY
		BaseURL:       getEnvOrDefault("NEWS_API_BASE_URL", "https://newsapi.org/v2/top-headlines"),
		Regions:       strings.Split(getEnvOrDefault("NEWS_REGIONS", "us,in,de"), ","),
		MaxPages:      getEnvIntOrDefault("NEWS_MAX_PAGES", 2),
		MaxArticles:   getEnvIntOrDefault("NEWS_MAX_ARTICLES", 50),
		RateLimit:     time.Duration(getEnvIntOrDefault("NEWS_RATE_LIMIT_SECONDS", 2)) * time.Second,
		FetchInterval: time.Duration(getEnvIntOrDefault("NEWS_FETCH_INTERVAL_HOURS", 2)) * time.Hour,
	}

	if config.APIKey == "" {
		log.Fatal("Missing NEWS_API_KEY environment variable")
	}

	log.Printf("News Config: BaseURL=%s, Regions=%v, MaxPages=%d, MaxArticles=%d",
		config.BaseURL, config.Regions, config.MaxPages, config.MaxArticles)

	return config
}

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

func fetchRegionNews(region string, config *NewsConfig) ([]model.Article, error) {
	// NewsAPI.org URL structure: https://newsapi.org/v2/top-headlines?country=us&apiKey=API_KEY&pageSize=20&page=1
	// GNews URL structure: https://gnews.io/api/v4/top-headlines?lang=en&country=us&max=10&token=API_KEY&page=1

	var baseURL string

	if strings.Contains(config.BaseURL, "newsapi.org") {
		// NewsAPI.org format
		baseURL = fmt.Sprintf("%s?country=%s&pageSize=20&apiKey=%s",
			config.BaseURL, region, config.APIKey)
	} else {
		// GNews format (fallback)
		baseURL = fmt.Sprintf("%s?lang=en&country=%s&max=10&token=%s",
			config.BaseURL, region, config.APIKey)
	}

	var allArticles []model.Article

	for page := 1; page <= config.MaxPages; page++ {
		url := fmt.Sprintf("%s&page=%d", baseURL, page)

		log.Printf("Fetching region=%s page=%d URL=%s", region, page, url)

		resp, err := http.Get(url)
		if err != nil {
			log.Printf("HTTP error for region=%s page=%d: %v", region, page, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("Non-200 response for region=%s page=%d: %s", region, page, resp.Status)
			continue
		}

		var result struct {
			Articles []model.Article `json:"articles"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			log.Printf("JSON decode error for region=%s page=%d: %v", region, page, err)
			continue
		}

		for _, article := range result.Articles {
			article.Topic = region
			article.FetchedAt = time.Now()
			allArticles = append(allArticles, article)
		}

		time.Sleep(config.RateLimit) // configurable rate limit
	}

	if len(allArticles) > config.MaxArticles {
		allArticles = allArticles[:config.MaxArticles]
	}
	log.Printf("Fetched %d articles for region=%s", len(allArticles), region)
	return allArticles, nil
}

func StartScheduledFetcher(db *mongo.Database) {
	config := loadNewsConfig()

	log.Println("Starting scheduled news fetcher...")

	// run immediately
	fetchAndStoreArticles(db, config)

	ticker := time.NewTicker(config.FetchInterval)
	for {
		<-ticker.C
		fetchAndStoreArticles(db, config)
	}
}

func fetchAndStoreArticles(db *mongo.Database, config *NewsConfig) {
	log.Println("Fetching news articles by region...")

	for _, region := range config.Regions {
		log.Printf("Region: %s", region)

		articles, err := fetchRegionNews(region, config)
		if err != nil {
			log.Printf("Failed to fetch news for region=%s: %v", region, err)
			continue
		}

		log.Printf("Inserting %d articles for region=%s", len(articles), region)

		for i, article := range articles {
			filter := bson.M{"url": article.URL}
			update := bson.M{"$set": article}
			_, err := db.Collection("articles").UpdateOne(context.TODO(), filter, update, options.Update().SetUpsert(true))
			if err != nil {
				log.Printf("[%d] Insert failed for article: %s | error: %v", i, article.URL, err)
			} else {
				log.Printf("[%d] Upserted article: %s", i, article.URL)
			}
		}
	}

	log.Println("Finished fetch and store cycle")
}
