package api

import (
	"analytics-service/handler"
	"log"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

func StartServer(db *mongo.Database) {
	r := gin.Default()

	// Enable CORS for all origins (you may want to restrict this in production)
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	r.Use(cors.New(config))

	// Create handlers
	analyticsHandler := handler.NewAnalyticsHandler(db)

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"service": "analytics-service",
		})
	})

	// Analytics endpoints
	api := r.Group("/api/v1")
	{
		api.POST("/analytics/track", analyticsHandler.TrackEvent)
		api.GET("/analytics/stats", analyticsHandler.GetStats)
	}

	log.Println("Analytics service starting on port 8080...")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
