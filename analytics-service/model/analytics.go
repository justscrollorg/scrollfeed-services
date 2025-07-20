package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// VisitEvent represents a user visit event
type VisitEvent struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	SessionID    string             `bson:"session_id" json:"session_id"`
	UserAgent    string             `bson:"user_agent" json:"user_agent"`
	IPAddress    string             `bson:"ip_address" json:"ip_address"`
	Referrer     string             `bson:"referrer" json:"referrer"`
	Page         string             `bson:"page" json:"page"`
	Timestamp    time.Time          `bson:"timestamp" json:"timestamp"`
	Country      string             `bson:"country,omitempty" json:"country,omitempty"`
	City         string             `bson:"city,omitempty" json:"city,omitempty"`
	Device       string             `bson:"device,omitempty" json:"device,omitempty"`
	Browser      string             `bson:"browser,omitempty" json:"browser,omitempty"`
	OS           string             `bson:"os,omitempty" json:"os,omitempty"`
	ScreenWidth  int                `bson:"screen_width,omitempty" json:"screen_width,omitempty"`
	ScreenHeight int                `bson:"screen_height,omitempty" json:"screen_height,omitempty"`
	Language     string             `bson:"language,omitempty" json:"language,omitempty"`
	TimeZone     string             `bson:"timezone,omitempty" json:"timezone,omitempty"`
}

// PageView represents a page view event
type PageView struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	SessionID   string             `bson:"session_id" json:"session_id"`
	Page        string             `bson:"page" json:"page"`
	Title       string             `bson:"title" json:"title"`
	URL         string             `bson:"url" json:"url"`
	Timestamp   time.Time          `bson:"timestamp" json:"timestamp"`
	TimeOnPage  int64              `bson:"time_on_page,omitempty" json:"time_on_page,omitempty"` // seconds
	ScrollDepth float64            `bson:"scroll_depth,omitempty" json:"scroll_depth,omitempty"` // percentage
	ExitPage    bool               `bson:"exit_page" json:"exit_page"`
}

// AnalyticsRequest represents the incoming analytics data from frontend
type AnalyticsRequest struct {
	SessionID    string  `json:"session_id" binding:"required"`
	UserAgent    string  `json:"user_agent"`
	Referrer     string  `json:"referrer"`
	Page         string  `json:"page" binding:"required"`
	Title        string  `json:"title"`
	URL          string  `json:"url" binding:"required"`
	ScreenWidth  int     `json:"screen_width"`
	ScreenHeight int     `json:"screen_height"`
	Language     string  `json:"language"`
	TimeZone     string  `json:"timezone"`
	TimeOnPage   int64   `json:"time_on_page"`
	ScrollDepth  float64 `json:"scroll_depth"`
	EventType    string  `json:"event_type"` // "visit", "pageview", "exit"
}

// AnalyticsStats represents aggregated analytics data
type AnalyticsStats struct {
	TotalVisits    int64            `json:"total_visits"`
	UniqueVisitors int64            `json:"unique_visitors"`
	PageViews      int64            `json:"page_views"`
	AvgTimeOnSite  float64          `json:"avg_time_on_site"`
	BounceRate     float64          `json:"bounce_rate"`
	TopPages       []PageStats      `json:"top_pages"`
	TopReferrers   []ReferrerStats  `json:"top_referrers"`
	DeviceTypes    map[string]int64 `json:"device_types"`
	Browsers       map[string]int64 `json:"browsers"`
	Countries      map[string]int64 `json:"countries"`
	HourlyStats    []HourlyStats    `json:"hourly_stats"`
}

type PageStats struct {
	Page    string  `json:"page"`
	Views   int64   `json:"views"`
	AvgTime float64 `json:"avg_time"`
}

type ReferrerStats struct {
	Referrer string `json:"referrer"`
	Visits   int64  `json:"visits"`
}

type HourlyStats struct {
	Hour   int   `json:"hour"`
	Visits int64 `json:"visits"`
}
