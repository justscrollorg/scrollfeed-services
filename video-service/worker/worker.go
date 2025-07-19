package worker

import (
	"context"
	"encoding/json"
	"log"
	"time"
	"video-service/config"
	"video-service/fetcher"
	"video-service/model"

	"github.com/nats-io/nats.go"
	"go.mongodb.org/mongo-driver/mongo"
)

type Worker struct {
	config     *config.Config
	natsConn   *nats.Conn
	fetcher    *fetcher.Fetcher
	cancelFunc context.CancelFunc
}

func NewWorker(cfg *config.Config, db *mongo.Database) (*Worker, error) {
	// Connect to NATS
	nc, err := nats.Connect(cfg.NATSUrl)
	if err != nil {
		return nil, err
	}

	// Create fetcher
	fetcher := fetcher.NewFetcher(cfg, db)

	return &Worker{
		config:   cfg,
		natsConn: nc,
		fetcher:  fetcher,
	}, nil
}

func (w *Worker) Start(ctx context.Context) error {
	log.Println("Starting video worker...")

	// Create cancellable context
	workerCtx, cancel := context.WithCancel(ctx)
	w.cancelFunc = cancel

	// Subscribe to video fetch requests
	_, err := w.natsConn.Subscribe("fetch.videos", func(msg *nats.Msg) {
		w.handleFetchRequest(workerCtx, msg)
	})
	if err != nil {
		return err
	}

	log.Println("Successfully subscribed to fetch.videos")

	// Start scheduler for periodic fetches
	go w.startScheduler(workerCtx)

	log.Println("Workers started successfully")
	return nil
}

func (w *Worker) Stop() {
	log.Println("Stopping video worker...")
	if w.cancelFunc != nil {
		w.cancelFunc()
	}
	if w.natsConn != nil {
		w.natsConn.Close()
	}
}

func (w *Worker) handleFetchRequest(ctx context.Context, msg *nats.Msg) {
	var req model.FetchRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		log.Printf("Failed to unmarshal fetch request: %v", err)
		return
	}

	log.Printf("Processing fetch request: %+v", req)

	// Process the fetch request
	result, err := w.fetcher.FetchVideos(ctx, req)
	if err != nil {
		log.Printf("Failed to fetch videos: %v", err)
		return
	}

	// Publish result if needed
	resultData, _ := json.Marshal(result)
	w.natsConn.Publish("fetch.videos.result", resultData)

	log.Printf("Completed fetch request: %s", req.RequestID)
}

func (w *Worker) startScheduler(ctx context.Context) {
	log.Println("Scheduler started on this instance")
	log.Println("Scheduling periodic video fetches")

	ticker := time.NewTicker(w.config.FetchInterval)
	defer ticker.Stop()

	// Define regions and categories to fetch
	regions := []string{"US", "IN", "DE", "GB", "CA"}
	categories := []string{"10", "24", "25"} // Music, Entertainment, News & Politics

	// Initial fetch
	w.scheduleVideoFetches(regions, categories)

	for {
		select {
		case <-ctx.Done():
			log.Println("Scheduler stopped")
			return
		case <-ticker.C:
			log.Println("Triggering scheduled video fetch")
			w.scheduleVideoFetches(regions, categories)
		}
	}
}

func (w *Worker) scheduleVideoFetches(regions, categories []string) {
	for _, region := range regions {
		for _, category := range categories {
			// Create fetch request
			req := model.FetchRequest{
				Region:    region,
				Category:  category,
				MaxVideos: 20,
				Priority:  "normal",
				RequestID: w.generateRequestID(region, category),
			}

			// Publish to NATS
			data, _ := json.Marshal(req)
			if err := w.natsConn.Publish("fetch.videos", data); err != nil {
				log.Printf("Failed to publish fetch request: %v", err)
			} else {
				log.Printf("Scheduled fetch for region %s, category %s", region, category)
			}

			// Rate limiting between requests
			time.Sleep(w.config.RateLimit)
		}
	}
}

func (w *Worker) generateRequestID(region, category string) string {
	timestamp := time.Now().Format("20060102-150405")
	return region + "-" + category + "-" + timestamp
}
