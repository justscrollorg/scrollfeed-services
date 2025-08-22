package api

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Region mapping for UI compatibility
func mapRegionToCode(region string) string {
	regionMap := map[string]string{
		"india":   "in",
		"in":      "in",
		"usa":     "us", 
		"us":      "us",
		"america": "us",
		"germany": "de",
		"de":      "de",
		"deutschland": "de",
	}
	
	normalized := strings.ToLower(strings.TrimSpace(region))
	if code, exists := regionMap[normalized]; exists {
		return code
	}
	return normalized // fallback to original
}

// Enhanced news handler with better pagination and caching
func enhancedNewsHandler(c *gin.Context, db *mongo.Database) {
	start := time.Now()

	// Parse query parameters
	region := mapRegionToCode(c.Query("region"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "33"))

	// Validate pagination
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 33
	}

	skip := (page - 1) * limit

	log.Printf("/news api called with region=%s, page=%d, limit=%d", region, page, limit)

	// Build filter
	filter := bson.M{}
	if region != "" {
		filter["topic"] = region
	}

	// Add timestamp-based consistency
	// Only show articles that were fetched before request started
	maxFetchTime := start.Add(-1 * time.Second) // 1 second buffer
	filter["fetchedAt"] = bson.M{"$lte": maxFetchTime}

	// Optimized aggregation pipeline for better performance
	pipeline := []bson.M{
		{"$match": filter},
		{"$sort": bson.M{"publishedAt": -1, "_id": 1}}, // Secondary sort for consistency
		{"$skip": skip},
		{"$limit": limit},
		{
			"$project": bson.M{
				"title":       1,
				"description": 1,
				"url":         1,
				"image":       1,
				"source":      1,
				"publishedAt": 1,
				"topic":       1,
				"fetchedAt":   1,
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := db.Collection("articles").Aggregate(ctx, pipeline)
	if err != nil {
		log.Printf("DB aggregation failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database query failed"})
		return
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		log.Printf("DB cursor decode failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Data processing failed"})
		return
	}

	// Get total count for pagination metadata (with same filter)
	totalCount, _ := db.Collection("articles").CountDocuments(ctx, filter)
	totalPages := (int(totalCount) + limit - 1) / limit

	// Response with pagination metadata
	response := gin.H{
		"articles": results,
		"metadata": gin.H{
			"page":         page,
			"limit":        limit,
			"total":        totalCount,
			"totalPages":   totalPages,
			"hasNext":      page < totalPages,
			"hasPrev":      page > 1,
			"region":       region,
			"responseTime": time.Since(start).String(),
		},
	}

	// Add cache headers for better client-side caching
	c.Header("Cache-Control", "public, max-age=300") // 5 minutes cache
	c.Header("Last-Modified", time.Now().UTC().Format(http.TimeFormat))

	log.Printf("Returned %d articles (page %d/%d) for region=%s in %v",
		len(results), page, totalPages, region, time.Since(start))

	c.JSON(http.StatusOK, response)
}

// Real-time stats endpoint for monitoring
func statsHandler(c *gin.Context, db *mongo.Database) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Aggregate statistics
	pipeline := []bson.M{
		{
			"$group": bson.M{
				"_id":         "$topic",
				"count":       bson.M{"$sum": 1},
				"lastFetched": bson.M{"$max": "$fetchedAt"},
			},
		},
		{
			"$sort": bson.M{"count": -1},
		},
	}

	cursor, err := db.Collection("articles").Aggregate(ctx, pipeline)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Stats query failed"})
		return
	}
	defer cursor.Close(ctx)

	var stats []bson.M
	cursor.All(ctx, &stats)

	c.JSON(http.StatusOK, gin.H{
		"regionStats": stats,
		"timestamp":   time.Now(),
	})
}
