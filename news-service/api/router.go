package api

import (
	"log"
	"news-service/handler"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

var dbmngo *mongo.Database
var natsNewsHandler *handler.NewsHandler

func StartServer(db *mongo.Database, natsURL string) {
	router := gin.Default()

	dbmngo = db

	// Initialize NATS handler
	var err error
	natsNewsHandler, err = handler.NewNewsHandler(db, natsURL)
	if err != nil {
		log.Fatal("Failed to initialize NATS handler:", err)
	}

	// Health check routes
	router.GET("/", healthCheck)
	router.GET("/health", healthCheck)
	router.GET("/ready", healthCheck)

	// API routes
	router.GET("/news-api/news", callnewsHandler)
	router.POST("/news-api/fetch/:region", triggerRegionFetch)
	router.POST("/news-api/fetch-all", triggerAllFetch)

	log.Println("News API is running at :8080")

	router.Run(":8080")
}

func healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{"status": "healthy", "service": "news-service"})
}

func callnewsHandler(c *gin.Context) {
	log.Printf("[INFO] callnewsHandler called")
	natsNewsHandler.GetNews(c, dbmngo)
}

func triggerRegionFetch(c *gin.Context) {
	region := c.Param("region")
	priority := c.DefaultQuery("priority", "normal")

	log.Printf("[INFO] Manual fetch triggered for region=%s, priority=%s", region, priority)

	if err := natsNewsHandler.TriggerNewsFetch(region, priority); err != nil {
		log.Printf("Failed to trigger fetch for region %s: %v", region, err)
		c.JSON(500, gin.H{"error": "Failed to trigger fetch", "details": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "Fetch triggered successfully", "region": region, "priority": priority})
}

func triggerAllFetch(c *gin.Context) {
	priority := c.DefaultQuery("priority", "high")

	log.Printf("[INFO] Manual fetch triggered for all regions, priority=%s", priority)

	if err := natsNewsHandler.TriggerAllRegionsFetch(priority); err != nil {
		log.Printf("Failed to trigger fetch for all regions: %v", err)
		c.JSON(500, gin.H{"error": "Failed to trigger fetch", "details": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "Fetch triggered for all regions", "priority": priority})
}
