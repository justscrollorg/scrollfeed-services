package handler

import (
	"log"
	"net/http"
	"video-service/service"

	"github.com/gin-gonic/gin"
)

func GetVideoStats(c *gin.Context) {
	videoID := c.Query("videoId")
	log.Printf("[INFO] GetVideoStats called with videoId: %s", videoID)

	if videoID == "" {
		log.Printf("[WARN] Missing videoId parameter in GetVideoStats request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "videoId is required"})
		return
	}

	stats, err := service.FetchVideoAndChannelStats(videoID)
	if err != nil {
		log.Printf("[ERROR] FetchVideoAndChannelStats failed for videoId=%s: %v", videoID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("[INFO] Successfully fetched stats for videoId=%s: Views=%s, Likes=%s, Comments=%s, Subscribers=%s",
		videoID, stats.ViewCount, stats.LikeCount, stats.CommentCount, stats.SubscriberCount)

	c.JSON(http.StatusOK, stats)
}
