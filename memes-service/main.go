package main

import (
	"context"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var mongoClient *mongo.Client
var memesCollection *mongo.Collection

func main() {
	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://mongo:27017"))
	if err != nil {
		log.Fatal(err)
	}
	mongoClient = client
	memesCollection = client.Database("memesdb").Collection("memes")

	go backgroundRefresh(ctx)

	r := gin.Default()
	r.GET("/memes/trending", getTrendingMemes)
	r.Run(":8080")
}

func backgroundRefresh(ctx context.Context) {
	for {
		refreshMemes(ctx)
		time.Sleep(30 * time.Minute)
	}
}

func refreshMemes(ctx context.Context) {
	imgflip, _ := FetchImgflipMemes()
	reddit, _ := FetchRedditMemes()
	memes := deduplicateMemes(append(imgflip, reddit...))
	if len(memes) == 0 {
		return
	}
	_, _ = memesCollection.DeleteMany(ctx, map[string]interface{}{})
	docs := []interface{}{}
	for _, m := range memes {
		docs = append(docs, m)
	}
	_, _ = memesCollection.InsertMany(ctx, docs)
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
	cursor, err := memesCollection.Find(ctx, map[string]interface{}{})
	if err != nil {
		c.JSON(500, gin.H{"error": "db error"})
		return
	}
	var memes []Meme
	if err := cursor.All(ctx, &memes); err != nil {
		c.JSON(500, gin.H{"error": "db error"})
		return
	}
	c.JSON(200, memes)
}
