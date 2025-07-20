package main

import (
	"analytics-service/api"
	"context"
	"log"
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

	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal("MongoDB connection error:", err)
	}

	// Test connection
	err = client.Ping(context.Background(), nil)
	if err != nil {
		log.Fatal("MongoDB ping error:", err)
	}

	log.Println("Connected to MongoDB successfully")

	db := client.Database("analyticsdb")

	log.Println("Starting the analytics service")
	api.StartServer(db)
}
