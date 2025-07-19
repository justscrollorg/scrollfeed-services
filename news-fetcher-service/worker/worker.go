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

	// Clean up any existing consumer with same name (from previous crashed instances)
	if err := cleanupConsumer(js, consumerName); err != nil {
		log.Printf("Warning: Failed to cleanup existing consumer %s: %v", consumerName, err)
	}

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
	log.Printf("Starting %d worker instances with consumer: %s", w.config.WorkerCount, w.consumerName)

	// Create consumer configuration with proper error handling
	consumerConfig := &nats.ConsumerConfig{
		Durable:        w.consumerName,
		AckPolicy:      nats.AckExplicitPolicy,
		MaxAckPending:  w.config.WorkerCount,
		AckWait:        30 * time.Second,
		MaxDeliver:     3,
		ReplayPolicy:   nats.ReplayInstantPolicy,
		FilterSubject:  "news.fetch.request",
	}

	// Create consumer with retry logic
	var sub *nats.Subscription
	var err error
	
	for attempts := 0; attempts < 3; attempts++ {
		// Try to create/update consumer
		_, err = w.js.AddConsumer("NEWS_FETCH", consumerConfig)
		if err != nil && !strings.Contains(err.Error(), "already exists") {
			log.Printf("Attempt %d: Failed to create consumer: %v", attempts+1, err)
			if attempts < 2 {
				time.Sleep(time.Duration(attempts+1) * time.Second)
				continue
			}
			return fmt.Errorf("failed to create consumer after 3 attempts: %v", err)
		}

		// Subscribe with ephemeral subscription for better cleanup
		sub, err = w.js.PullSubscribe("news.fetch.request", w.consumerName, nats.ManualAck())
		if err != nil {
			if strings.Contains(err.Error(), "already bound") {
				log.Printf("Attempt %d: Consumer already bound, cleaning up and retrying...", attempts+1)
				cleanupConsumer(w.js, w.consumerName)
				time.Sleep(time.Duration(attempts+1) * time.Second)
				continue
			}
			return fmt.Errorf("failed to subscribe: %v", err)
		}
		break
	}

	if sub == nil {
		return fmt.Errorf("failed to create subscription after 3 attempts")
	}

	// Start message processing in goroutine
	go w.processMessages(ctx, sub)

	// Start scheduler for periodic fetches
	go w.startScheduler(ctx)

	log.Println("Workers started successfully")

	// Wait for context cancellation
	<-ctx.Done()
	
	// Cleanup on shutdown
	log.Printf("Shutting down worker %s, cleaning up consumer...", w.instanceID)
	if err := cleanupConsumer(w.js, w.consumerName); err != nil {
		log.Printf("Failed to cleanup consumer on shutdown: %v", err)
	}
	
	return ctx.Err()
}

func (w *Worker) processMessages(ctx context.Context, sub *nats.Subscription) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Fetch messages in batches
			msgs, err := sub.Fetch(1, nats.MaxWait(5*time.Second))
			if err != nil {
				if err == nats.ErrTimeout {
					continue // No messages available, continue polling
				}
				log.Printf("Error fetching messages: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}

			for _, msg := range msgs {
				w.handleFetchRequest(msg)
			}
		}
	}
}

func (w *Worker) handleFetchRequest(msg *nats.Msg) {
	var req model.FetchRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		log.Printf("Failed to unmarshal fetch request: %v", err)
		msg.Nak()
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
		msg.Nak()
		return
	}

	// Publish success result
	w.publishResult(result)
	msg.Ack()
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

func cleanupConsumer(js nats.JetStreamContext, consumerName string) error {
	// Try to delete the consumer, ignore errors if it doesn't exist
	err := js.DeleteConsumer("NEWS_FETCH", consumerName)
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return err
	}
	log.Printf("Cleaned up consumer: %s", consumerName)
	return nil
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
