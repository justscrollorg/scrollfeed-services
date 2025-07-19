package worker

import (
	"context"
	"encoding/json"
	"log"
	"news-fetcher-service/config"
	"news-fetcher-service/fetcher"
	"news-fetcher-service/model"
	"time"

	"github.com/nats-io/nats.go"
	"go.mongodb.org/mongo-driver/mongo"
)

type Worker struct {
	config  *config.Config
	nc      *nats.Conn
	fetcher *fetcher.Fetcher
	js      nats.JetStreamContext
}

func NewWorker(cfg *config.Config, nc *nats.Conn, db *mongo.Database) (*Worker, error) {
	js, err := nc.JetStream()
	if err != nil {
		return nil, err
	}

	// Create streams and consumers
	if err := setupStreams(js); err != nil {
		return nil, err
	}

	return &Worker{
		config:  cfg,
		nc:      nc,
		fetcher: fetcher.NewFetcher(cfg, db),
		js:      js,
	}, nil
}

func (w *Worker) Start(ctx context.Context) error {
	log.Printf("Starting %d worker instances", w.config.WorkerCount)

	// Subscribe to fetch requests
	_, err := w.js.Subscribe("news.fetch.request", w.handleFetchRequest,
		nats.Durable("news-fetcher-workers"),
		nats.ManualAck(),
		nats.MaxAckPending(w.config.WorkerCount),
	)
	if err != nil {
		return err
	}

	// Start scheduler for periodic fetches
	go w.startScheduler(ctx)

	log.Println("Workers started successfully")

	// Wait for context cancellation
	<-ctx.Done()
	return ctx.Err()
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
