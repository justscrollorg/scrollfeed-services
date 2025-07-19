package main

import (
	"context"
	"log"
	"net/http"
	"news-fetcher-service/config"
	"news-fetcher-service/worker"
	"os"
	"os/signal"
	"syscall"

	"github.com/nats-io/nats.go"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	log.Println("Starting News Fetcher Service...")

	// Load configuration
	cfg := config.Load()

	// Connect to MongoDB
	mongoClient, err := mongo.Connect(context.Background(), options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}
	defer mongoClient.Disconnect(context.Background())

	db := mongoClient.Database("newsdb")
	log.Println("Connected to MongoDB")

	// Connect to NATS
	nc, err := nats.Connect(cfg.NATSUrl)
	if err != nil {
		log.Fatal("Failed to connect to NATS:", err)
	}
	defer nc.Close()
	log.Println("Connected to NATS")

	// Create and start worker
	w, err := worker.NewWorker(cfg, nc, db)
	if err != nil {
		log.Fatal("Failed to create worker:", err)
	}

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal, stopping...")
		cancel()
	}()

	// Start health check server
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","service":"news-fetcher-service"}`))
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","service":"news-fetcher-service"}`))
	})

	http.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready","service":"news-fetcher-service"}`))
	})

	go func() {
		log.Println("Health check server starting on :8080")
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Printf("Health check server error: %v", err)
		}
	}()

	// Start worker
	log.Println("News fetcher service is running...")
	if err := w.Start(ctx); err != nil && err != context.Canceled {
		log.Fatal("Worker failed:", err)
	}

	log.Println("News fetcher service stopped")
}
