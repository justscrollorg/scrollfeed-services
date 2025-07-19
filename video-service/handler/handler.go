package handler

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"
	"video-service/model"
	"video-service/service"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var db *mongo.Database

func InitDB(database *mongo.Database) {
	db = database
}

func GetCategories(c *gin.Context) {
	region := c.DefaultQuery("region", "US")
	log.Printf("[INFO] GetCategories called with region: %s", region)

	// For simplicity, return predefined categories
	categories := []model.CategoryResponse{
		{ID: "10", Title: "Music"},
		{ID: "24", Title: "Entertainment"},
		{ID: "25", Title: "News & Politics"},
		{ID: "17", Title: "Sports"},
		{ID: "20", Title: "Gaming"},
		{ID: "23", Title: "Comedy"},
		{ID: "26", Title: "Howto & Style"},
		{ID: "27", Title: "Education"},
		{ID: "28", Title: "Science & Technology"},
	}

	log.Printf("[INFO] Retrieved %d categories for region %s", len(categories), region)
	c.JSON(http.StatusOK, categories)
}

func GetRegions(c *gin.Context) {
	log.Printf("[INFO] GetRegions called")

	// Return supported regions with names
	regions := []model.RegionResponse{
		{Code: "US", Name: "United States"},
		{Code: "IN", Name: "India"},
		{Code: "DE", Name: "Germany"},
		{Code: "GB", Name: "United Kingdom"},
		{Code: "CA", Name: "Canada"},
	}

	log.Printf("[INFO] Retrieved %d regions", len(regions))
	c.JSON(http.StatusOK, regions)
}

func GetVideos(c *gin.Context) {
	region := c.Query("region")
	category := c.Query("category")
	maxResultsStr := c.DefaultQuery("maxResults", "20")
	pageStr := c.DefaultQuery("page", "1")

	log.Printf("[INFO] GetVideos called with region: %s, category: %s, maxResults: %s, page: %s",
		region, category, maxResultsStr, pageStr)

	if region == "" {
		log.Printf("[WARN] Missing region parameter")
		c.JSON(http.StatusBadRequest, gin.H{"error": "region is required"})
		return
	}

	maxResults, err := strconv.Atoi(maxResultsStr)
	if err != nil || maxResults <= 0 || maxResults > 50 {
		log.Printf("[WARN] Invalid maxResults: %s", maxResultsStr)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid maxResults, must be between 1 and 50"})
		return
	}

	page, err := strconv.Atoi(pageStr)
	if err != nil || page <= 0 {
		page = 1
	}

	// Build filter
	filter := bson.M{"region": region}
	if category != "" && category != "0" {
		filter["categoryId"] = category
	}

	// Calculate skip for pagination
	skip := (page - 1) * maxResults

	// Query options
	opts := options.Find().
		SetSort(bson.D{{Key: "publishedAt", Value: -1}}).
		SetLimit(int64(maxResults)).
		SetSkip(int64(skip))

	// Execute query
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := db.Collection("videos")
	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		log.Printf("[ERROR] Database query failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database query failed"})
		return
	}
	defer cursor.Close(ctx)

	var videos []model.Video
	if err := cursor.All(ctx, &videos); err != nil {
		log.Printf("[ERROR] Failed to decode videos: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode videos"})
		return
	}

	// Transform videos to YouTube API format for frontend compatibility
	transformedVideos := make([]map[string]interface{}, len(videos))
	for i, video := range videos {
		transformedVideos[i] = map[string]interface{}{
			"id": map[string]interface{}{
				"videoId": video.VideoID,
			},
			"snippet": map[string]interface{}{
				"title":        video.Title,
				"description":  video.Description,
				"channelTitle": video.ChannelTitle,
				"publishedAt":  video.PublishedAt.Format(time.RFC3339),
				"thumbnails": map[string]interface{}{
					"medium": map[string]interface{}{
						"url": video.Thumbnail,
					},
					"default": map[string]interface{}{
						"url": video.Thumbnail,
					},
				},
				"categoryId": video.CategoryID,
			},
			"videoURL":     video.VideoURL,
			"viewCount":    video.ViewCount,
			"likeCount":    video.LikeCount,
			"duration":     video.Duration,
			"region":       video.Region,
			"categoryName": video.CategoryName,
		}
	}

	log.Printf("[INFO] Retrieved %d videos for region=%s, category=%s", len(videos), region, category)
	c.JSON(http.StatusOK, transformedVideos)
}

func GetTrending(c *gin.Context) {
	maxResultsStr := c.DefaultQuery("maxResults", "20")
	region := c.DefaultQuery("region", "US")

	log.Printf("[INFO] GetTrending called with maxResults=%s, region=%s", maxResultsStr, region)

	maxResults, err := strconv.Atoi(maxResultsStr)
	if err != nil || maxResults <= 0 || maxResults > 50 {
		log.Printf("[WARN] Invalid maxResults: %s", maxResultsStr)
		c.JSON(http.StatusBadRequest, gin.H{"error": "maxResults must be between 1 and 50"})
		return
	}

	// Get trending videos (sorted by view count)
	filter := bson.M{"region": region}
	opts := options.Find().
		SetSort(bson.D{{Key: "viewCount", Value: -1}}).
		SetLimit(int64(maxResults))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := db.Collection("videos")
	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		log.Printf("[ERROR] Database query failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database query failed"})
		return
	}
	defer cursor.Close(ctx)

	var videos []model.Video
	if err := cursor.All(ctx, &videos); err != nil {
		log.Printf("[ERROR] Failed to decode videos: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode videos"})
		return
	}

	// Transform videos to YouTube API format for frontend compatibility
	transformedVideos := make([]map[string]interface{}, len(videos))
	for i, video := range videos {
		transformedVideos[i] = map[string]interface{}{
			"id": map[string]interface{}{
				"videoId": video.VideoID,
			},
			"snippet": map[string]interface{}{
				"title":        video.Title,
				"description":  video.Description,
				"channelTitle": video.ChannelTitle,
				"publishedAt":  video.PublishedAt.Format(time.RFC3339),
				"thumbnails": map[string]interface{}{
					"medium": map[string]interface{}{
						"url": video.Thumbnail,
					},
					"default": map[string]interface{}{
						"url": video.Thumbnail,
					},
				},
				"categoryId": video.CategoryID,
			},
			"videoURL":     video.VideoURL,
			"viewCount":    video.ViewCount,
			"likeCount":    video.LikeCount,
			"duration":     video.Duration,
			"region":       video.Region,
			"categoryName": video.CategoryName,
		}
	}

	log.Printf("[INFO] Retrieved %d trending videos for region=%s", len(videos), region)
	c.JSON(http.StatusOK, transformedVideos)
}

func SearchVideos(c *gin.Context) {
	query := c.Query("query")
	region := c.DefaultQuery("region", "US")
	maxResultsStr := c.DefaultQuery("maxResults", "10")

	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter 'query' is required"})
		return
	}

	maxResults, err := strconv.Atoi(maxResultsStr)
	if err != nil || maxResults <= 0 || maxResults > 50 {
		maxResults = 10
	}

	// Search in title and description
	filter := bson.M{
		"region": region,
		"$or": []bson.M{
			{"title": bson.M{"$regex": query, "$options": "i"}},
			{"description": bson.M{"$regex": query, "$options": "i"}},
		},
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "publishedAt", Value: -1}}).
		SetLimit(int64(maxResults))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := db.Collection("videos")
	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		log.Printf("[ERROR] Search query failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Search failed"})
		return
	}
	defer cursor.Close(ctx)

	var videos []model.Video
	if err := cursor.All(ctx, &videos); err != nil {
		log.Printf("[ERROR] Failed to decode search results: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode search results"})
		return
	}

	// Transform videos to YouTube API format for frontend compatibility
	transformedVideos := make([]map[string]interface{}, len(videos))
	for i, video := range videos {
		transformedVideos[i] = map[string]interface{}{
			"id": map[string]interface{}{
				"videoId": video.VideoID,
			},
			"snippet": map[string]interface{}{
				"title":        video.Title,
				"description":  video.Description,
				"channelTitle": video.ChannelTitle,
				"publishedAt":  video.PublishedAt.Format(time.RFC3339),
				"thumbnails": map[string]interface{}{
					"medium": map[string]interface{}{
						"url": video.Thumbnail,
					},
					"default": map[string]interface{}{
						"url": video.Thumbnail,
					},
				},
				"categoryId": video.CategoryID,
			},
			"videoURL":     video.VideoURL,
			"viewCount":    video.ViewCount,
			"likeCount":    video.LikeCount,
			"duration":     video.Duration,
			"region":       video.Region,
			"categoryName": video.CategoryName,
		}
	}

	log.Printf("[INFO] Search returned %d videos for query='%s', region=%s", len(videos), query, region)
	c.JSON(http.StatusOK, transformedVideos)
}

// Legacy handlers that still use YouTube API directly for compatibility
func GetComments(c *gin.Context) {
	videoID := c.Query("videoId")
	maxResultsStr := c.DefaultQuery("maxResults", "10")

	log.Printf("[INFO] GetComments called with videoId: %s, maxResults: %s", videoID, maxResultsStr)

	if videoID == "" {
		log.Printf("[WARN] Missing videoId")
		c.JSON(http.StatusBadRequest, gin.H{"error": "videoId is required"})
		return
	}

	maxResults, err := strconv.Atoi(maxResultsStr)
	if err != nil || maxResults <= 0 || maxResults > 100 {
		log.Printf("[WARN] Invalid maxResults: %s", maxResultsStr)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid maxResults, must be between 1 and 100"})
		return
	}

	data, err := service.FetchComments(videoID, maxResults)
	if err != nil {
		log.Printf("[ERROR] FetchComments failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("[INFO] Retrieved comments for videoId=%s", videoID)
	c.JSON(http.StatusOK, data)
}

func GetVideoStats(c *gin.Context) {
	videoID := c.Query("videoId")
	log.Printf("[INFO] GetVideoStats called with videoId: %s", videoID)

	if videoID == "" {
		log.Printf("[WARN] Missing videoId")
		c.JSON(http.StatusBadRequest, gin.H{"error": "videoId is required"})
		return
	}

	data, err := service.FetchVideoStatistics(videoID)
	if err != nil {
		log.Printf("[ERROR] FetchVideoStatistics failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("[INFO] Retrieved video statistics for videoId=%s", videoID)
	c.JSON(http.StatusOK, data)
}
