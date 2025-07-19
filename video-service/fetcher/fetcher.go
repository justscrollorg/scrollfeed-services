package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
	"video-service/config"
	"video-service/model"

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

	collection := f.db.Collection("videos")

	// Compound index for optimal query performance
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "region", Value: 1},
				{Key: "categoryId", Value: 1},
				{Key: "publishedAt", Value: -1},
			},
		},
		{
			Keys:    bson.D{{Key: "videoId", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "fetchedAt", Value: -1}},
		},
	}

	for _, index := range indexes {
		_, err := collection.Indexes().CreateOne(ctx, index)
		if err != nil {
			log.Printf("Warning: Failed to create index: %v", err)
		}
	}
}

func (f *Fetcher) FetchVideos(ctx context.Context, req model.FetchRequest) (model.FetchResult, error) {
	result := model.FetchResult{
		RequestID:   req.RequestID,
		ProcessedAt: time.Now(),
	}

	log.Printf("Fetching videos for region=%s, category=%s, maxVideos=%d, requestID=%s",
		req.Region, req.Category, req.MaxVideos, req.RequestID)

	// Fetch trending videos for the region and category
	videos, err := f.fetchTrendingVideos(ctx, req.Region, req.Category, req.MaxVideos)
	if err != nil {
		result.Error = err.Error()
		return result, err
	}

	if len(videos) == 0 {
		result.Error = "No videos fetched"
		return result, fmt.Errorf("no videos fetched for region %s, category %s", req.Region, req.Category)
	}

	// Store videos in MongoDB
	stored, err := f.storeVideos(ctx, videos, req.Region, req.Category)
	if err != nil {
		result.Error = err.Error()
		return result, err
	}

	result.Success = true
	result.VideosCount = stored
	log.Printf("Successfully processed %d videos for region=%s, category=%s, requestID=%s",
		stored, req.Region, req.Category, req.RequestID)

	return result, nil
}

func (f *Fetcher) fetchTrendingVideos(ctx context.Context, region, categoryID string, maxResults int) ([]model.Video, error) {
	// Build YouTube API URL for trending videos
	url := fmt.Sprintf("https://www.googleapis.com/youtube/v3/videos?part=snippet,statistics,contentDetails&chart=mostPopular&regionCode=%s&maxResults=%d&key=%s",
		region, maxResults, f.config.YouTubeAPIKey)

	if categoryID != "" && categoryID != "0" {
		url += fmt.Sprintf("&videoCategoryId=%s", categoryID)
	}

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
		return nil, fmt.Errorf("YouTube API HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	var apiResponse model.YouTubeVideoResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, err
	}

	// Convert to our model format
	var videos []model.Video
	now := time.Now()

	for _, apiVideo := range apiResponse.Items {
		// Parse published date
		publishedAt, _ := time.Parse(time.RFC3339, apiVideo.Snippet.PublishedAt)

		// Get the best thumbnail
		thumbnail := f.getBestThumbnail(apiVideo.Snippet.Thumbnails)

		// Parse view count
		viewCount, _ := strconv.ParseInt(apiVideo.Statistics.ViewCount, 10, 64)
		likeCount, _ := strconv.ParseInt(apiVideo.Statistics.LikeCount, 10, 64)

		// Get category name
		categoryName := f.getCategoryName(apiVideo.Snippet.CategoryID)

		video := model.Video{
			VideoID:      apiVideo.ID,
			Title:        apiVideo.Snippet.Title,
			Description:  apiVideo.Snippet.Description,
			ChannelTitle: apiVideo.Snippet.ChannelTitle,
			CategoryID:   apiVideo.Snippet.CategoryID,
			CategoryName: categoryName,
			Region:       region,
			PublishedAt:  publishedAt,
			Thumbnail:    thumbnail,
			VideoURL:     fmt.Sprintf("https://www.youtube.com/watch?v=%s", apiVideo.ID),
			ViewCount:    viewCount,
			LikeCount:    likeCount,
			Duration:     apiVideo.ContentDetails.Duration,
			FetchedAt:    now,
			Source: struct {
				Name string `json:"name" bson:"name"`
			}{
				Name: "YouTube",
			},
		}

		videos = append(videos, video)
	}

	return videos, nil
}

func (f *Fetcher) getBestThumbnail(thumbnails model.Thumbnails) string {
	if thumbnails.High.URL != "" {
		return thumbnails.High.URL
	}
	if thumbnails.Medium.URL != "" {
		return thumbnails.Medium.URL
	}
	if thumbnails.Default.URL != "" {
		return thumbnails.Default.URL
	}
	return "https://via.placeholder.com/320x180/E5E7EB/6B7280?text=Video+Thumbnail"
}

func (f *Fetcher) getCategoryName(categoryID string) string {
	categoryMap := map[string]string{
		"1":  "Film & Animation",
		"2":  "Autos & Vehicles",
		"10": "Music",
		"15": "Pets & Animals",
		"17": "Sports",
		"19": "Travel & Events",
		"20": "Gaming",
		"22": "People & Blogs",
		"23": "Comedy",
		"24": "Entertainment",
		"25": "News & Politics",
		"26": "Howto & Style",
		"27": "Education",
		"28": "Science & Technology",
	}

	if name, exists := categoryMap[categoryID]; exists {
		return name
	}
	return "General"
}

func (f *Fetcher) storeVideos(ctx context.Context, videos []model.Video, region, category string) (int, error) {
	if len(videos) == 0 {
		return 0, nil
	}

	collection := f.db.Collection("videos")

	var operations []mongo.WriteModel
	for _, video := range videos {
		// Use upsert to handle duplicates
		filter := bson.M{"videoId": video.VideoID}
		update := bson.M{
			"$set": bson.M{
				"title":        video.Title,
				"description":  video.Description,
				"channelTitle": video.ChannelTitle,
				"categoryId":   video.CategoryID,
				"categoryName": video.CategoryName,
				"region":       video.Region,
				"publishedAt":  video.PublishedAt,
				"thumbnail":    video.Thumbnail,
				"videoUrl":     video.VideoURL,
				"viewCount":    video.ViewCount,
				"likeCount":    video.LikeCount,
				"duration":     video.Duration,
				"fetchedAt":    video.FetchedAt,
				"source":       video.Source,
			},
		}

		operation := mongo.NewUpdateOneModel().
			SetFilter(filter).
			SetUpdate(update).
			SetUpsert(true)

		operations = append(operations, operation)
	}

	// Execute bulk write
	result, err := collection.BulkWrite(ctx, operations)
	if err != nil {
		return 0, err
	}

	stored := int(result.UpsertedCount + result.ModifiedCount)
	log.Printf("Bulk operation completed: %d upserted, %d modified, %d total processed",
		result.UpsertedCount, result.ModifiedCount, len(operations))

	return stored, nil
}

func (f *Fetcher) buildYouTubeURL(region, category string, maxResults int) string {
	baseURL := "https://www.googleapis.com/youtube/v3/videos"
	params := fmt.Sprintf("?part=snippet,statistics,contentDetails&chart=mostPopular&regionCode=%s&maxResults=%d&key=%s",
		region, maxResults, f.config.YouTubeAPIKey)

	if category != "" && category != "0" {
		params += fmt.Sprintf("&videoCategoryId=%s", category)
	}

	return baseURL + params
}
