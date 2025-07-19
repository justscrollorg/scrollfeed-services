package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type NewsHandler struct {
	db *mongo.Database
	nc *nats.Conn
	js nats.JetStreamContext
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

func NewNewsHandler(db *mongo.Database, natsURL string) (*NewsHandler, error) {
	nc, err := nats.Connect(natsURL)
	if err != nil {
		return nil, err
	}

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, err
	}

	handler := &NewsHandler{
		db: db,
		nc: nc,
		js: js,
	}

	// Subscribe to fetch results for monitoring
	go handler.subscribeFetchResults()

	return handler, nil
}

func (h *NewsHandler) Close() {
	if h.nc != nil {
		h.nc.Close()
	}
}

func (h *NewsHandler) TriggerNewsFetch(region string, priority string) error {
	req := FetchRequest{
		Region:    region,
		MaxPages:  4,
		Priority:  priority,
		RequestID: generateRequestID(region),
	}

	data, err := json.Marshal(req)
	if err != nil {
		return err
	}

	log.Printf("Triggering manual fetch for region=%s, priority=%s", region, priority)
	_, err = h.js.Publish("news.fetch.request", data)
	return err
}

func (h *NewsHandler) TriggerAllRegionsFetch(priority string) error {
	regions := []string{"us", "in", "de"}

	for _, region := range regions {
		if err := h.TriggerNewsFetch(region, priority); err != nil {
			log.Printf("Failed to trigger fetch for region %s: %v", region, err)
			return err
		}
	}

	log.Printf("Triggered fetch for all regions with priority=%s", priority)
	return nil
}

func (h *NewsHandler) subscribeFetchResults() {
	_, err := h.js.Subscribe("news.fetch.result", func(msg *nats.Msg) {
		var result FetchResult
		if err := json.Unmarshal(msg.Data, &result); err != nil {
			log.Printf("Failed to unmarshal fetch result: %v", err)
			return
		}

		if result.Success {
			log.Printf("Fetch completed successfully: region=%s, articles=%d, requestID=%s",
				result.Region, result.ArticleCount, result.RequestID)
		} else {
			log.Printf("Fetch failed: region=%s, error=%s, requestID=%s",
				result.Region, result.Error, result.RequestID)
		}

		msg.Ack()
	}, nats.Durable("news-service-results"))

	if err != nil {
		log.Printf("Failed to subscribe to fetch results: %v", err)
	}
}

func generateRequestID(region string) string {
	return region + "-manual-" + time.Now().Format("20060102-150405")
}

func (h *NewsHandler) GetNews(c *gin.Context, db *mongo.Database) {
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
		return
	}
	defer cursor.Close(context.TODO())

	var results []bson.M
	if err := cursor.All(context.TODO(), &results); err != nil {
		log.Printf("DB cursor decode failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("Returned %d articles for region=%s in %v", len(results), region, time.Since(start))
	c.JSON(http.StatusOK, results)
}
