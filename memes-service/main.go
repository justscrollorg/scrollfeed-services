package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var mongoClient *mongo.Client
var memesCollection *mongo.Collection

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
	memesCollection = client.Database("memesdb").Collection("memes")

	go backgroundRefresh(ctx)

	r := gin.Default()
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "timestamp": time.Now()})
	})
	r.GET("/memes/trending", getTrendingMemes)
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
	for {
		log.Println("Refreshing memes in background...")
		refreshMemes(ctx)
		time.Sleep(30 * time.Minute)
	}
}

func refreshMemes(ctx context.Context) {
	imgflip, err1 := FetchImgflipMemes()
	if err1 != nil {
		log.Printf("Error fetching Imgflip memes: %v", err1)
	}
	reddit, err2 := FetchRedditMemes()
	if err2 != nil {
		log.Printf("Error fetching Reddit memes: %v", err2)
	}
	memes := deduplicateMemes(append(imgflip, reddit...))
	if len(memes) == 0 {
		log.Println("No memes fetched from sources.")
		return
	}
	if memesCollection == nil {
		log.Println("memesCollection is nil, skipping DB update")
		return
	}
	_, err := memesCollection.DeleteMany(ctx, map[string]interface{}{})
	if err != nil {
		log.Printf("Error deleting old memes: %v", err)
	}
	docs := []interface{}{}
	for _, m := range memes {
		docs = append(docs, m)
	}
	_, err = memesCollection.InsertMany(ctx, docs)
	if err != nil {
		log.Printf("Error inserting memes: %v", err)
	} else {
		log.Printf("Inserted %d memes", len(memes))
	}
}

func deduplicateMemes(memes []Meme) []Meme {
	seen := map[string]bool{}
	result := []Meme{}
	for _, m := range memes {
		if !seen[m.ImageURL] {
			seen[m.ImageURL] = true
			result = append(result, m)
		}
	}
	return result
}

func getTrendingMemes(c *gin.Context) {
	ctx := context.Background()
	if memesCollection == nil {
		log.Println("memesCollection is nil in handler")
		c.JSON(500, gin.H{"error": "db not ready"})
		return
	}
	cursor, err := memesCollection.Find(ctx, map[string]interface{}{})
	if err != nil {
		log.Printf("DB Find error: %v", err)
		c.JSON(500, gin.H{"error": "db error"})
		return
	}
	var memes []Meme
	if err := cursor.All(ctx, &memes); err != nil {
		log.Printf("DB cursor.All error: %v", err)
		c.JSON(500, gin.H{"error": "db error"})
		return
	}
	log.Printf("Serving %d memes", len(memes))
	c.JSON(200, memes)
}
