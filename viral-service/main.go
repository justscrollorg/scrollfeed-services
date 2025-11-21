package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var mongoClient *mongo.Client
var viralCollection *mongo.Collection

func main() {
	ctx := context.Background()
	mongoURI := getenv("MONGO_URI", "mongodb://mongo:27017")

	log.Printf("Connecting to MongoDB at %s", mongoURI)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("MongoDB connect error: %v", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf("MongoDB ping error: %v", err)
	}

	log.Println("MongoDB connection successful")
	mongoClient = client
	viralCollection = client.Database("viraldb").Collection("stories")

	// Start background refresh
	go backgroundRefresh(ctx)

	// Setup Gin router
	r := gin.Default()

	// Health check endpoints
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "timestamp": time.Now()})
	})

	r.GET("/ready", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ready", "timestamp": time.Now()})
	})

	// API routes with /viral-api prefix to match ingress routing
	api := r.Group("/viral-api")
	{
		api.GET("/trending", getTrendingViral)
		api.GET("/refresh", triggerRefresh)
		api.GET("/sources", getSources)
		api.GET("/categories", getCategories)
	}

	// Also keep the original routes for direct access
	r.GET("/viral/trending", getTrendingViral)
	r.GET("/viral/refresh", triggerRefresh)

	log.Println("Starting viral service on :8080")
	r.Run(":8080")
}

func getenv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

func backgroundRefresh(ctx context.Context) {
	// Initial refresh on startup
	log.Println("Performing initial viral stories refresh...")
	refreshViral(ctx)

	// Refresh every 20 minutes
	ticker := time.NewTicker(20 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Println("Refreshing viral stories in background...")
			refreshViral(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func refreshViral(ctx context.Context) {
	log.Println("Starting viral stories fetch...")

	// Fetch from Reddit
	redditStories, err1 := FetchRedditViral()
	if err1 != nil {
		log.Printf("Error fetching Reddit viral stories: %v", err1)
	} else {
		log.Printf("Fetched %d stories from Reddit", len(redditStories))
	}

	// Fetch from HackerNews
	hnStories, err2 := FetchHackerNewsViral()
	if err2 != nil {
		log.Printf("Error fetching HackerNews viral stories: %v", err2)
	} else {
		log.Printf("Fetched %d stories from HackerNews", len(hnStories))
	}

	// Combine all stories
	allStories := append(redditStories, hnStories...)

	// Deduplicate
	stories := deduplicateStories(allStories)

	if len(stories) == 0 {
		log.Println("No viral stories fetched from any source.")
		return
	}

	// Sort by viral score
	sort.Slice(stories, func(i, j int) bool {
		return stories[i].ViralScore > stories[j].ViralScore
	})

	// Keep top 100 stories
	if len(stories) > 100 {
		stories = stories[:100]
	}

	if viralCollection == nil {
		log.Println("viralCollection is nil, skipping DB update")
		return
	}

	// Clear old stories
	_, err := viralCollection.DeleteMany(ctx, bson.M{})
	if err != nil {
		log.Printf("Error deleting old viral stories: %v", err)
	}

	// Insert new stories
	docs := []interface{}{}
	for _, s := range stories {
		docs = append(docs, s)
	}

	_, err = viralCollection.InsertMany(ctx, docs)
	if err != nil {
		log.Printf("Error inserting viral stories: %v", err)
	} else {
		log.Printf("Inserted %d viral stories into database", len(stories))
	}
}

func getTrendingViral(c *gin.Context) {
	ctx := context.Background()

	if viralCollection == nil {
		log.Println("viralCollection is nil in handler")
		c.JSON(500, gin.H{"error": "database not ready"})
		return
	}

	// Get query parameters
	source := c.DefaultQuery("source", "")
	category := c.DefaultQuery("category", "")
	limit := c.DefaultQuery("limit", "50")

	limitInt := 50
	if l, err := parseInt(limit); err == nil && l > 0 {
		limitInt = l
		if limitInt > 100 {
			limitInt = 100
		}
	}

	// Build filter
	filter := bson.M{}
	if source != "" {
		filter["source"] = source
	}
	if category != "" {
		filter["category"] = category
	}

	// Find options: sort by viral score descending
	opts := options.Find().
		SetSort(bson.D{{Key: "viral_score", Value: -1}}).
		SetLimit(int64(limitInt))

	cursor, err := viralCollection.Find(ctx, filter, opts)
	if err != nil {
		log.Printf("DB Find error: %v", err)
		c.JSON(500, gin.H{"error": "database error"})
		return
	}

	var stories []ViralStory
	if err := cursor.All(ctx, &stories); err != nil {
		log.Printf("DB cursor.All error: %v", err)
		c.JSON(500, gin.H{"error": "database error"})
		return
	}

	log.Printf("Serving %d viral stories (source: %s, category: %s)", len(stories), source, category)

	c.JSON(200, gin.H{
		"count":   len(stories),
		"stories": stories,
	})
}

func triggerRefresh(c *gin.Context) {
	ctx := context.Background()

	log.Println("Manual refresh triggered via API")
	go refreshViral(ctx)

	c.JSON(200, gin.H{
		"message": "Refresh triggered",
		"status":  "processing",
	})
}

func getSources(c *gin.Context) {
	sources := []string{"reddit", "hackernews"}
	c.JSON(200, gin.H{
		"sources": sources,
	})
}

func getCategories(c *gin.Context) {
	categories := []string{"worldnews", "news", "technology"}
	c.JSON(200, gin.H{
		"categories": categories,
	})
}

func parseInt(s string) (int, error) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}
