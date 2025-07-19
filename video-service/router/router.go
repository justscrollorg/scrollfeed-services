package router

import (
	"github.com/gin-gonic/gin"
	"video-service/handler"
)

func Setup() *gin.Engine {
	r := gin.Default()

	r.GET("/api/regions", handler.GetRegions)
	r.GET("/api/categories", handler.GetCategories)
	r.GET("/api/videos", handler.GetTopVideos)
	r.GET("/api/trending", handler.GetAllTrending)
	r.GET("/api/search", handler.SearchVideos)
	r.GET("/api/comments", handler.GetComments)
	r.GET("/api/videostats", handler.GetVideoStats)

	return r
}
