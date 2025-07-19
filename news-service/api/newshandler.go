package api

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func newsHandler(c *gin.Context, db *mongo.Database) bool {
	start := time.Now()
	region := strings.ToLower(c.Query("region"))  
	log.Printf("/news api called with region=%s", region)

	filter := bson.M{}
	if region != "" {
		filter["topic"] = region
	}

	opts := options.Find().
		SetSort(bson.M{"publishedAt": -1}).
		SetLimit(33)

	log.Printf("Querying DB with filter: %v", filter)

	cursor, err := db.Collection("articles").Find(context.TODO(), filter, opts)
	if err != nil {
		log.Printf("DB query failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return true
	}
	defer cursor.Close(context.TODO())

	var results []bson.M
	if err := cursor.All(context.TODO(), &results); err != nil {
		log.Printf("DB cursor decode failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return true
	}

	log.Printf("Returned %d articles for region=%s in %v", len(results), region, time.Since(start))
	c.JSON(http.StatusOK, results)
	return false
}
