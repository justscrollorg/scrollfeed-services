package router

import (
	"video-service/handler"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

func Setup(db *mongo.Database) *gin.Engine {
	r := gin.Default()

	// CORS middleware
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"*"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// Initialize handlers with database
	handler.InitDB(db)

	r.GET("/api/regions", handler.GetRegions)
	r.GET("/api/categories", handler.GetCategories)
	r.GET("/api/videos", handler.GetVideos)
	r.GET("/api/trending", handler.GetTrending)
	r.GET("/api/search", handler.SearchVideos)
	r.GET("/api/comments", handler.GetComments)
	r.GET("/api/videostats", handler.GetVideoStats)

	// Health check endpoint
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "video-service"})
	})
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "service": "video-service"})
	})

	return r
}
