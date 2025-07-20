package handler

import (
	"analytics-service/model"
	"context"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type AnalyticsHandler struct {
	db *mongo.Database
}

func NewAnalyticsHandler(db *mongo.Database) *AnalyticsHandler {
	return &AnalyticsHandler{db: db}
}

// TrackEvent handles incoming analytics events
func (h *AnalyticsHandler) TrackEvent(c *gin.Context) {
	var req model.AnalyticsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get client IP address
	clientIP := h.getClientIP(c)

	// Process based on event type
	switch req.EventType {
	case "visit":
		h.recordVisit(c, req, clientIP)
	case "pageview":
		h.recordPageView(c, req)
	case "exit":
		h.recordExit(c, req)
	default:
		// Default to visit event
		h.recordVisit(c, req, clientIP)
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Event tracked successfully",
	})
}

func (h *AnalyticsHandler) recordVisit(c *gin.Context, req model.AnalyticsRequest, clientIP string) {
	visit := model.VisitEvent{
		SessionID:    req.SessionID,
		UserAgent:    req.UserAgent,
		IPAddress:    clientIP,
		Referrer:     req.Referrer,
		Page:         req.Page,
		Timestamp:    time.Now(),
		Device:       h.parseDevice(req.UserAgent),
		Browser:      h.parseBrowser(req.UserAgent),
		OS:           h.parseOS(req.UserAgent),
		ScreenWidth:  req.ScreenWidth,
		ScreenHeight: req.ScreenHeight,
		Language:     req.Language,
		TimeZone:     req.TimeZone,
	}

	// Insert visit event
	collection := h.db.Collection("visits")
	_, err := collection.InsertOne(context.Background(), visit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record visit"})
		return
	}
}

func (h *AnalyticsHandler) recordPageView(c *gin.Context, req model.AnalyticsRequest) {
	pageView := model.PageView{
		SessionID:   req.SessionID,
		Page:        req.Page,
		Title:       req.Title,
		URL:         req.URL,
		Timestamp:   time.Now(),
		TimeOnPage:  req.TimeOnPage,
		ScrollDepth: req.ScrollDepth,
		ExitPage:    false,
	}

	collection := h.db.Collection("pageviews")
	_, err := collection.InsertOne(context.Background(), pageView)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record pageview"})
		return
	}
}

func (h *AnalyticsHandler) recordExit(c *gin.Context, req model.AnalyticsRequest) {
	// Update the last page view to mark as exit page
	collection := h.db.Collection("pageviews")
	filter := bson.M{
		"session_id": req.SessionID,
		"page":       req.Page,
	}
	update := bson.M{
		"$set": bson.M{
			"exit_page":    true,
			"time_on_page": req.TimeOnPage,
			"scroll_depth": req.ScrollDepth,
		},
	}

	opts := options.FindOneAndUpdate().SetSort(bson.D{primitive.E{Key: "timestamp", Value: -1}})
	err := collection.FindOneAndUpdate(context.Background(), filter, update, opts).Err()
	if err != nil {
		// If no existing page view found, create one
		pageView := model.PageView{
			SessionID:   req.SessionID,
			Page:        req.Page,
			Title:       req.Title,
			URL:         req.URL,
			Timestamp:   time.Now(),
			TimeOnPage:  req.TimeOnPage,
			ScrollDepth: req.ScrollDepth,
			ExitPage:    true,
		}
		collection.InsertOne(context.Background(), pageView)
	}
}

// GetStats returns analytics statistics
func (h *AnalyticsHandler) GetStats(c *gin.Context) {
	days := c.DefaultQuery("days", "7")
	daysInt, _ := strconv.Atoi(days)

	startDate := time.Now().AddDate(0, 0, -daysInt)

	stats, err := h.calculateStats(startDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *AnalyticsHandler) calculateStats(startDate time.Time) (*model.AnalyticsStats, error) {
	ctx := context.Background()

	// Total visits
	visitsCollection := h.db.Collection("visits")
	totalVisits, _ := visitsCollection.CountDocuments(ctx, bson.M{
		"timestamp": bson.M{"$gte": startDate},
	})

	// Unique visitors (by session_id)
	pipeline := []bson.M{
		{"$match": bson.M{"timestamp": bson.M{"$gte": startDate}}},
		{"$group": bson.M{"_id": "$session_id"}},
		{"$count": "unique_visitors"},
	}
	cursor, _ := visitsCollection.Aggregate(ctx, pipeline)
	var uniqueResult []bson.M
	cursor.All(ctx, &uniqueResult)

	var uniqueVisitors int64 = 0
	if len(uniqueResult) > 0 {
		if val, ok := uniqueResult[0]["unique_visitors"].(int32); ok {
			uniqueVisitors = int64(val)
		}
	}

	// Page views
	pageViewsCollection := h.db.Collection("pageviews")
	totalPageViews, _ := pageViewsCollection.CountDocuments(ctx, bson.M{
		"timestamp": bson.M{"$gte": startDate},
	})

	// Top pages
	topPagesPipeline := []bson.M{
		{"$match": bson.M{"timestamp": bson.M{"$gte": startDate}}},
		{"$group": bson.M{
			"_id":      "$page",
			"views":    bson.M{"$sum": 1},
			"avg_time": bson.M{"$avg": "$time_on_page"},
		}},
		{"$sort": bson.M{"views": -1}},
		{"$limit": 10},
	}

	cursor, _ = pageViewsCollection.Aggregate(ctx, topPagesPipeline)
	var topPagesResult []bson.M
	cursor.All(ctx, &topPagesResult)

	var topPages []model.PageStats
	for _, page := range topPagesResult {
		pageStats := model.PageStats{
			Page:    page["_id"].(string),
			Views:   int64(page["views"].(int32)),
			AvgTime: 0,
		}
		if avgTime, ok := page["avg_time"]; ok && avgTime != nil {
			pageStats.AvgTime = avgTime.(float64)
		}
		topPages = append(topPages, pageStats)
	}

	stats := &model.AnalyticsStats{
		TotalVisits:    totalVisits,
		UniqueVisitors: uniqueVisitors,
		PageViews:      totalPageViews,
		TopPages:       topPages,
		DeviceTypes:    make(map[string]int64),
		Browsers:       make(map[string]int64),
		Countries:      make(map[string]int64),
	}

	return stats, nil
}

func (h *AnalyticsHandler) getClientIP(c *gin.Context) string {
	// Check for X-Forwarded-For header first (for load balancers/proxies)
	xff := c.GetHeader("X-Forwarded-For")
	if xff != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check for X-Real-IP header
	realIP := c.GetHeader("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		return c.Request.RemoteAddr
	}
	return ip
}

func (h *AnalyticsHandler) parseDevice(userAgent string) string {
	ua := strings.ToLower(userAgent)
	if strings.Contains(ua, "mobile") || strings.Contains(ua, "android") || strings.Contains(ua, "iphone") {
		return "mobile"
	} else if strings.Contains(ua, "tablet") || strings.Contains(ua, "ipad") {
		return "tablet"
	}
	return "desktop"
}

func (h *AnalyticsHandler) parseBrowser(userAgent string) string {
	ua := strings.ToLower(userAgent)
	if strings.Contains(ua, "chrome") {
		return "Chrome"
	} else if strings.Contains(ua, "firefox") {
		return "Firefox"
	} else if strings.Contains(ua, "safari") {
		return "Safari"
	} else if strings.Contains(ua, "edge") {
		return "Edge"
	}
	return "Unknown"
}

func (h *AnalyticsHandler) parseOS(userAgent string) string {
	ua := strings.ToLower(userAgent)
	if strings.Contains(ua, "windows") {
		return "Windows"
	} else if strings.Contains(ua, "mac") {
		return "macOS"
	} else if strings.Contains(ua, "linux") {
		return "Linux"
	} else if strings.Contains(ua, "android") {
		return "Android"
	} else if strings.Contains(ua, "ios") {
		return "iOS"
	}
	return "Unknown"
}
