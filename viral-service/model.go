package main

import "time"

// ViralStory represents a viral news story from multiple sources
type ViralStory struct {
	ID          string    `json:"id" bson:"_id,omitempty"`
	Title       string    `json:"title" bson:"title"`
	Description string    `json:"description" bson:"description"`
	URL         string    `json:"url" bson:"url"`
	ImageURL    string    `json:"image_url" bson:"image_url"`
	Source      string    `json:"source" bson:"source"` // "reddit", "hackernews", "twitter"
	SourceID    string    `json:"source_id" bson:"source_id"`
	Author      string    `json:"author" bson:"author"`
	Category    string    `json:"category" bson:"category"`
	PublishedAt time.Time `json:"published_at" bson:"published_at"`
	FetchedAt   time.Time `json:"fetched_at" bson:"fetched_at"`

	// Viral metrics
	ViralScore int     `json:"viral_score" bson:"viral_score"`
	Upvotes    int     `json:"upvotes" bson:"upvotes"`
	Comments   int     `json:"comments" bson:"comments"`
	Shares     int     `json:"shares" bson:"shares"`
	Engagement float64 `json:"engagement" bson:"engagement"`
}

// RedditPost represents a post from Reddit API
type RedditPost struct {
	Data struct {
		Title       string  `json:"title"`
		Selftext    string  `json:"selftext"`
		URL         string  `json:"url"`
		Permalink   string  `json:"permalink"`
		Author      string  `json:"author"`
		Subreddit   string  `json:"subreddit"`
		Thumbnail   string  `json:"thumbnail"`
		Score       int     `json:"score"`
		NumComments int     `json:"num_comments"`
		Created     float64 `json:"created_utc"`
		ID          string  `json:"id"`
	} `json:"data"`
}

// RedditResponse represents the Reddit API response
type RedditResponse struct {
	Data struct {
		Children []RedditPost `json:"children"`
	} `json:"data"`
}

// HackerNewsItem represents a story from HackerNews API
type HackerNewsItem struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	By          string `json:"by"`
	Score       int    `json:"score"`
	Descendants int    `json:"descendants"` // number of comments
	Time        int64  `json:"time"`
	Type        string `json:"type"`
}
