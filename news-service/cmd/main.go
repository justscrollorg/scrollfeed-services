package main

import (
	"context"
	"log"
	"news-service/api"
	"os"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		log.Fatal("MONGO_URI is not set")
	} else {
		log.Printf("MONGO_URI is set: %s", mongoURI)
	}

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
		log.Printf("NATS_URL not set, using default: %s", natsURL)
	} else {
		log.Printf("NATS_URL is set: %s", natsURL)
	}

	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal("MongoDB connection error:", err)
	}

	db := client.Database("newsdb")

	log.Println("Starting the news service (API only)")
	api.StartServer(db, natsURL)
}
