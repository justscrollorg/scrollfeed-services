package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"news-fetcher-service/config"
	"news-fetcher-service/model"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Fetcher struct {
	config *config.Config
	db     *mongo.Database
	client *http.Client
}

func NewFetcher(cfg *config.Config, db *mongo.Database) *Fetcher {
	f := &Fetcher{
		config: cfg,
		db:     db,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Ensure optimal indexes for read performance
	f.ensureIndexes()
	return f
}

func (f *Fetcher) ensureIndexes() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := f.db.Collection("articles")

	// Compound index for optimal query performance
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "topic", Value: 1},
				{Key: "publishedAt", Value: -1},
				{Key: "fetchedAt", Value: -1},
			},
		},
		{
			Keys:    bson.D{{Key: "url", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "publishedAt", Value: -1}},
		},
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		log.Printf("Warning: Failed to create indexes: %v", err)
	} else {
		log.Println("Database indexes ensured for optimal performance")
	}
}

func (f *Fetcher) FetchRegionNews(ctx context.Context, req model.FetchRequest) (*model.FetchResult, error) {
	log.Printf("Fetching news for region=%s, maxPages=%d, requestID=%s", req.Region, req.MaxPages, req.RequestID)

	result := &model.FetchResult{
		Region:    req.Region,
		RequestID: req.RequestID,
		FetchedAt: time.Now(),
		Success:   false,
	}

	var allArticles []model.Article

	// Support both NewsAPI.org and GNews formats
	var baseURL string
	if f.config.NewsAPIBaseURL == "https://newsapi.org/v2/top-headlines" ||
		strings.Contains(f.config.NewsAPIBaseURL, "newsapi.org") {
		// NewsAPI.org format with region-specific handling
		baseURL = f.buildNewsAPIURL(req.Region)
	} else {
		// GNews format (fallback)
		baseURL = fmt.Sprintf("%s?lang=en&country=%s&max=10&token=%s",
			f.config.NewsAPIBaseURL, req.Region, f.config.NewsAPIKey)
	}

	for page := 1; page <= req.MaxPages; page++ {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		url := fmt.Sprintf("%s&page=%d", baseURL, page)
		articles, err := f.fetchPage(ctx, url, req.Region)
		if err != nil {
			log.Printf("Failed to fetch page %d for region %s: %v", page, req.Region, err)
			// Continue with other pages instead of failing completely
			continue
		}

		allArticles = append(allArticles, articles...)

		// Rate limiting between requests
		if page < req.MaxPages {
			time.Sleep(f.config.RateLimit)
		}
	}

	if len(allArticles) == 0 {
		result.Error = "No articles fetched"
		return result, fmt.Errorf("no articles fetched for region %s", req.Region)
	}

	// Limit articles to prevent overwhelming the database
	if len(allArticles) > 33 {
		allArticles = allArticles[:33]
	}

	// Store articles in database
	storedCount, err := f.storeArticles(ctx, allArticles)
	if err != nil {
		result.Error = err.Error()
		return result, err
	}

	result.Success = true
	result.ArticleCount = storedCount
	log.Printf("Successfully processed %d articles for region=%s, requestID=%s", storedCount, req.Region, req.RequestID)

	return result, nil
}

func (f *Fetcher) fetchPage(ctx context.Context, url, region string) ([]model.Article, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	var result struct {
		Articles []model.Article `json:"articles"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	// Set region and fetch time for all articles
	now := time.Now()
	for i := range result.Articles {
		result.Articles[i].Topic = region
		result.Articles[i].FetchedAt = now
	}

	return result.Articles, nil
}

func (f *Fetcher) storeArticles(ctx context.Context, articles []model.Article) (int, error) {
	if len(articles) == 0 {
		return 0, nil
	}

	collection := f.db.Collection("articles")

	// Use bulk operations for better performance
	var operations []mongo.WriteModel

	for _, article := range articles {
		// Use ReplaceOne with upsert for atomic updates
		operation := mongo.NewReplaceOneModel().
			SetFilter(bson.M{"url": article.URL}).
			SetReplacement(article).
			SetUpsert(true)

		operations = append(operations, operation)
	}

	// Configure bulk write options for performance
	opts := options.BulkWrite().
		SetOrdered(false). // Allow parallel processing
		SetBypassDocumentValidation(false)

	result, err := collection.BulkWrite(ctx, operations, opts)
	if err != nil {
		log.Printf("Bulk write failed: %v", err)
		return 0, err
	}

	storedCount := int(result.UpsertedCount + result.ModifiedCount)
	log.Printf("Bulk operation completed: %d upserted, %d modified, %d total processed",
		result.UpsertedCount, result.ModifiedCount, storedCount)

	return storedCount, nil
}

func (f *Fetcher) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return f.db.Client().Ping(ctx, nil)
}

// buildNewsAPIURL creates region-specific NewsAPI.org URLs
func (f *Fetcher) buildNewsAPIURL(region string) string {
	switch region {
	case "us":
		// US works well with country parameter
		return fmt.Sprintf("%s?country=us&pageSize=20&apiKey=%s",
			f.config.NewsAPIBaseURL, f.config.NewsAPIKey)
	case "in":
		// India: Use specific sources for better results
		return fmt.Sprintf("%s?sources=the-times-of-india,the-hindu&pageSize=20&apiKey=%s",
			f.config.NewsAPIBaseURL, f.config.NewsAPIKey)
	case "de":
		// Germany: Use specific sources for better results
		return fmt.Sprintf("%s?sources=spiegel-online,der-tagesspiegel,focus&pageSize=20&apiKey=%s",
			f.config.NewsAPIBaseURL, f.config.NewsAPIKey)
	case "gb", "uk":
		// UK: Use country parameter
		return fmt.Sprintf("%s?country=gb&pageSize=20&apiKey=%s",
			f.config.NewsAPIBaseURL, f.config.NewsAPIKey)
	case "ca":
		// Canada: Use country parameter
		return fmt.Sprintf("%s?country=ca&pageSize=20&apiKey=%s",
			f.config.NewsAPIBaseURL, f.config.NewsAPIKey)
	case "au":
		// Australia: Use country parameter
		return fmt.Sprintf("%s?country=au&pageSize=20&apiKey=%s",
			f.config.NewsAPIBaseURL, f.config.NewsAPIKey)
	default:
		// Fallback: try country parameter
		return fmt.Sprintf("%s?country=%s&pageSize=20&apiKey=%s",
			f.config.NewsAPIBaseURL, region, f.config.NewsAPIKey)
	}
}
