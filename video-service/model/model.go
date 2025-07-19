package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Video represents a video stored in MongoDB
type Video struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	VideoID      string             `bson:"videoId" json:"videoId"`
	Title        string             `bson:"title" json:"title"`
	Description  string             `bson:"description" json:"description"`
	ChannelTitle string             `bson:"channelTitle" json:"channelTitle"`
	CategoryID   string             `bson:"categoryId" json:"categoryId"`
	CategoryName string             `bson:"categoryName" json:"categoryName"`
	Region       string             `bson:"region" json:"region"`
	PublishedAt  time.Time          `bson:"publishedAt" json:"publishedAt"`
	Thumbnail    string             `bson:"thumbnail" json:"thumbnail"`
	VideoURL     string             `bson:"videoUrl" json:"videoUrl"`
	ViewCount    int64              `bson:"viewCount" json:"viewCount"`
	LikeCount    int64              `bson:"likeCount" json:"likeCount"`
	Duration     string             `bson:"duration" json:"duration"`
	FetchedAt    time.Time          `bson:"fetchedAt" json:"fetchedAt"`
	Source       struct {
		Name string `json:"name" bson:"name"`
	} `json:"source" bson:"source"`
}

// FetchRequest represents a video fetch request via NATS
type FetchRequest struct {
	Region    string `json:"region"`
	Category  string `json:"category"`
	MaxVideos int    `json:"maxVideos"`
	Priority  string `json:"priority"`
	RequestID string `json:"requestId"`
}

// FetchResult represents the result of a video fetch operation
type FetchResult struct {
	Success     bool      `json:"success"`
	VideosCount int       `json:"videosCount"`
	RequestID   string    `json:"requestId"`
	Error       string    `json:"error,omitempty"`
	ProcessedAt time.Time `json:"processedAt"`
}

// YouTube API Response structures
type YouTubeVideoResponse struct {
	Items []YouTubeVideoItem `json:"items"`
}

type YouTubeVideoItem struct {
	ID      string `json:"id"`
	Snippet struct {
		Title        string     `json:"title"`
		Description  string     `json:"description"`
		ChannelTitle string     `json:"channelTitle"`
		CategoryID   string     `json:"categoryId"`
		PublishedAt  string     `json:"publishedAt"`
		Thumbnails   Thumbnails `json:"thumbnails"`
	} `json:"snippet"`
	Statistics struct {
		ViewCount string `json:"viewCount"`
		LikeCount string `json:"likeCount"`
	} `json:"statistics"`
	ContentDetails struct {
		Duration string `json:"duration"`
	} `json:"contentDetails"`
}

type YouTubeTrendingResponse struct {
	Items []YouTubeTrendingItem `json:"items"`
}

type YouTubeTrendingItem struct {
	ID struct {
		VideoID string `json:"videoId"`
	} `json:"id"`
	Snippet struct {
		Title        string     `json:"title"`
		Description  string     `json:"description"`
		ChannelTitle string     `json:"channelTitle"`
		CategoryID   string     `json:"categoryId"`
		PublishedAt  string     `json:"publishedAt"`
		Thumbnails   Thumbnails `json:"thumbnails"`
	} `json:"snippet"`
}

type CategoryListResponse struct {
	Items []struct {
		ID      string `json:"id"`
		Snippet struct {
			Title      string `json:"title"`
			Assignable bool   `json:"assignable"`
		} `json:"snippet"`
	} `json:"items"`
}

type RegionListResponse struct {
	Items []struct {
		ID      string `json:"id"`
		Snippet struct {
			Name string `json:"name"`
		} `json:"snippet"`
	} `json:"items"`
}

type Thumbnail struct {
	URL    string `json:"url"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

type Thumbnails struct {
	Default  Thumbnail `json:"default"`
	Medium   Thumbnail `json:"medium"`
	High     Thumbnail `json:"high"`
	Standard Thumbnail `json:"standard"`
	Maxres   Thumbnail `json:"maxres"`
}

// Response structures for API
type VideoListResponse struct {
	Videos []Video `json:"videos"`
	Count  int     `json:"count"`
	Region string  `json:"region"`
}

type CategoryResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type RegionResponse struct {
	Code string `json:"code"`
	Name string `json:"name"`
}
