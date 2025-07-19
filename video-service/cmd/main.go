package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"video-service/config"
	"video-service/router"
	"video-service/worker"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Connect to MongoDB
	mongoClient, err := mongo.Connect(context.Background(), options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}
	defer mongoClient.Disconnect(context.Background())

	db := mongoClient.Database("videosdb")

	// Setup router with database connection
	r := router.Setup(db)

	// Create and start worker
	videoWorker, err := worker.NewWorker(cfg, db)
	if err != nil {
		log.Fatal("Failed to create worker:", err)
	}

	// Start worker in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := videoWorker.Start(ctx); err != nil {
		log.Fatal("Failed to start worker:", err)
	}

	// Setup HTTP server
	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	// Start server in background
	go func() {
		log.Println("Video service starting on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to start server:", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down video service...")

	// Graceful shutdown
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	videoWorker.Stop()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Video service stopped")
}
