package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"news-fetcher-service/config"
	"news-fetcher-service/fetcher"
	"news-fetcher-service/model"
	"os"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"go.mongodb.org/mongo-driver/mongo"
)

type Worker struct {
	config       *config.Config
	nc           *nats.Conn
	fetcher      *fetcher.Fetcher
	js           nats.JetStreamContext
	consumerName string
	instanceID   string
}

func NewWorker(cfg *config.Config, nc *nats.Conn, db *mongo.Database) (*Worker, error) {
	js, err := nc.JetStream()
	if err != nil {
		return nil, err
	}

	// Create unique instance ID for this worker
	instanceID := generateInstanceID()
	consumerName := fmt.Sprintf("news-fetcher-%s", instanceID)

	log.Printf("Creating worker with instanceID: %s, consumerName: %s", instanceID, consumerName)

	// Create streams and clean up any existing consumer with same name
	if err := setupStreams(js); err != nil {
		return nil, err
	}

	// Don't cleanup existing shared consumer since other instances may be using it
	log.Printf("Using existing shared consumer: news-fetcher-workers")

	return &Worker{
		config:       cfg,
		nc:           nc,
		fetcher:      fetcher.NewFetcher(cfg, db),
		js:           js,
		consumerName: consumerName,
		instanceID:   instanceID,
	}, nil
}

func (w *Worker) Start(ctx context.Context) error {
	log.Printf("Starting %d worker instances with consumer: news-fetcher-workers", w.config.WorkerCount)

	// Use simple async subscription without any JetStream complexity
	log.Println("Creating simple async subscription for news.fetch.request")

	// Use regular NATS subscription instead of JetStream pull subscription
	_, err := w.nc.Subscribe("news.fetch.request", func(msg *nats.Msg) {
		w.handleFetchRequest(msg)
	})
	if err != nil {
		return fmt.Errorf("failed to create async subscription: %v", err)
	}

	log.Printf("Successfully subscribed to news.fetch.request")

	// Start scheduler for periodic fetches (only run on first instance)
	if w.shouldRunScheduler() {
		go w.startScheduler(ctx)
		log.Println("Scheduler started on this instance")
	} else {
		log.Println("Scheduler skipped - another instance is likely running it")
	}

	log.Println("Workers started successfully")

	// Wait for context cancellation
	<-ctx.Done()

	log.Printf("Shutting down worker %s...", w.instanceID)
	return ctx.Err()
}

func (w *Worker) handleFetchRequest(msg *nats.Msg) {
	var req model.FetchRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		log.Printf("Failed to unmarshal fetch request: %v", err)
		return
	}

	log.Printf("Processing fetch request: %+v", req)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	result, err := w.fetcher.FetchRegionNews(ctx, req)
	if err != nil {
		log.Printf("Fetch failed for region %s: %v", req.Region, err)
		// Publish failure result
		w.publishResult(result)
		return
	}

	// Publish success result
	w.publishResult(result)
}

func (w *Worker) publishResult(result *model.FetchResult) {
	data, err := json.Marshal(result)
	if err != nil {
		log.Printf("Failed to marshal fetch result: %v", err)
		return
	}

	_, err = w.js.Publish("news.fetch.result", data)
	if err != nil {
		log.Printf("Failed to publish fetch result: %v", err)
	}
}

func (w *Worker) startScheduler(ctx context.Context) {
	regions := []string{"us", "in", "de"}
	ticker := time.NewTicker(w.config.FetchInterval)
	defer ticker.Stop()

	// Run immediately on startup
	w.scheduleRegionFetches(regions)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.scheduleRegionFetches(regions)
		}
	}
}

func (w *Worker) scheduleRegionFetches(regions []string) {
	log.Println("Scheduling periodic news fetches")

	for _, region := range regions {
		req := model.FetchRequest{
			Region:    region,
			MaxPages:  4,
			Priority:  "normal",
			RequestID: generateRequestID(region),
		}

		data, err := json.Marshal(req)
		if err != nil {
			log.Printf("Failed to marshal fetch request for region %s: %v", region, err)
			continue
		}

		_, err = w.js.Publish("news.fetch.request", data)
		if err != nil {
			log.Printf("Failed to schedule fetch for region %s: %v", region, err)
		} else {
			log.Printf("Scheduled fetch for region %s", region)
		}
	}
}

func generateRequestID(region string) string {
	return region + "-" + time.Now().Format("20060102-150405")
}

func generateInstanceID() string {
	// Try to get hostname first
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	// Add timestamp for uniqueness
	timestamp := time.Now().Format("150405")

	// Clean hostname to make it NATS-compatible
	hostname = strings.ReplaceAll(hostname, "-", "")
	hostname = strings.ReplaceAll(hostname, ".", "")

	return fmt.Sprintf("%s-%s", hostname, timestamp)
}

func (w *Worker) shouldRunScheduler() bool {
	// Simple leadership election - only the first instance (lexicographically) runs the scheduler
	// This prevents multiple schedulers from running at the same time
	hostname, _ := os.Hostname()
	return strings.HasPrefix(hostname, "news-fetcher-service") || hostname == "unknown"
}

func setupStreams(js nats.JetStreamContext) error {
	// Create stream for fetch requests
	_, err := js.AddStream(&nats.StreamConfig{
		Name:      "NEWS_FETCH",
		Subjects:  []string{"news.fetch.>"},
		Retention: nats.WorkQueuePolicy,
		MaxAge:    24 * time.Hour,
		Storage:   nats.FileStorage,
	})
	if err != nil && err != nats.ErrStreamNameAlreadyInUse {
		return err
	}

	log.Println("NATS streams configured successfully")
	return nil
}
