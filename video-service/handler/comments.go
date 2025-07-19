package handler

import (
	"log"
	"net/http"
	"video-service/service"

	"github.com/gin-gonic/gin"
)

func GetComments(c *gin.Context) {
	videoID := c.Query("videoId")
	log.Printf("[INFO] GetComments called with videoId: %s", videoID)

	if videoID == "" {
		log.Printf("[WARN] Missing videoId parameter in GetComments request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "videoId is required"})
		return
	}

	comments, err := service.FetchAllComments(videoID)
	if err != nil {
		log.Printf("[ERROR] FetchAllComments failed for videoId=%s: %v", videoID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("[INFO] Successfully fetched %d comments (including replies) for videoId=%s", len(comments), videoID)
	c.JSON(http.StatusOK, comments)
}
