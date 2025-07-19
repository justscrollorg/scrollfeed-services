package model

import "time"

type Article struct {
	Title       string `json:"title" bson:"title"`
	Description string `json:"description" bson:"description"`
	URL         string `json:"url" bson:"url"`
	Image       string `json:"image" bson:"image"`
	Source      struct {
		Name string `json:"name" bson:"name"`
	} `json:"source" bson:"source"`
	PublishedAt time.Time `json:"publishedAt" bson:"publishedAt"`
	Topic       string    `json:"topic" bson:"topic"`
	FetchedAt   time.Time `json:"fetchedAt" bson:"fetchedAt"`
}

type FetchRequest struct {
	Region    string `json:"region"`
	MaxPages  int    `json:"maxPages"`
	Priority  string `json:"priority"` // "high", "normal", "low"
	RequestID string `json:"requestId"`
}

type FetchResult struct {
	Region       string    `json:"region"`
	ArticleCount int       `json:"articleCount"`
	Success      bool      `json:"success"`
	Error        string    `json:"error,omitempty"`
	FetchedAt    time.Time `json:"fetchedAt"`
	RequestID    string    `json:"requestId"`
}
