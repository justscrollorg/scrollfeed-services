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
	DeviceModel  string             `bson:"device_model,omitempty" json:"device_model,omitempty"`
	Browser      string             `bson:"browser,omitempty" json:"browser,omitempty"`
	BrowserVersion string           `bson:"browser_version,omitempty" json:"browser_version,omitempty"`
	OS           string             `bson:"os,omitempty" json:"os,omitempty"`
	OSVersion    string             `bson:"os_version,omitempty" json:"os_version,omitempty"`
	ScreenWidth  int                `bson:"screen_width,omitempty" json:"screen_width,omitempty"`
	ScreenHeight int                `bson:"screen_height,omitempty" json:"screen_height,omitempty"`
	Language     string             `bson:"language,omitempty" json:"language,omitempty"`
	TimeZone     string             `bson:"timezone,omitempty" json:"timezone,omitempty"`
	// Additional device metadata
	CPUCores               int     `bson:"cpu_cores,omitempty" json:"cpu_cores,omitempty"`
	DeviceMemory          float64 `bson:"device_memory,omitempty" json:"device_memory,omitempty"`
	ConnectionType        string  `bson:"connection_type,omitempty" json:"connection_type,omitempty"`
	ConnectionDownlink    float64 `bson:"connection_downlink,omitempty" json:"connection_downlink,omitempty"`
	Platform              string  `bson:"platform,omitempty" json:"platform,omitempty"`
	Vendor                string  `bson:"vendor,omitempty" json:"vendor,omitempty"`
	TouchSupport          bool    `bson:"touch_support,omitempty" json:"touch_support,omitempty"`
	ColorDepth            int     `bson:"color_depth,omitempty" json:"color_depth,omitempty"`
	PixelRatio            float64 `bson:"pixel_ratio,omitempty" json:"pixel_ratio,omitempty"`
	AvailableScreenWidth  int     `bson:"available_screen_width,omitempty" json:"available_screen_width,omitempty"`
	AvailableScreenHeight int     `bson:"available_screen_height,omitempty" json:"available_screen_height,omitempty"`
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
	// Additional device metadata
	CPUCores               int     `json:"cpu_cores"`
	DeviceMemory          float64 `json:"device_memory"`
	ConnectionType        string  `json:"connection_type"`
	ConnectionDownlink    float64 `json:"connection_downlink"`
	Platform              string  `json:"platform"`
	Vendor                string  `json:"vendor"`
	TouchSupport          bool    `json:"touch_support"`
	ColorDepth            int     `json:"color_depth"`
	PixelRatio            float64 `json:"pixel_ratio"`
	AvailableScreenWidth  int     `json:"available_screen_width"`
	AvailableScreenHeight int     `json:"available_screen_height"`
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
