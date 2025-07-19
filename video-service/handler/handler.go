package handler

import (
	"log"
	"net/http"
	"video-service/service"
	"video-service/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

func GetCategories(c *gin.Context) {
	region := c.DefaultQuery("region", "US")
	log.Printf("[INFO] GetCategories called with region: %s", region)

	data, err := service.FetchCategories(region)
	if err != nil {
		log.Printf("[ERROR] FetchCategories failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(data) == 0 {
		log.Printf("[WARN] No categories returned for region: %s", region)
	}

	log.Printf("[INFO] Retrieved %d categories for region %s", len(data), region)
	c.JSON(http.StatusOK, data)
}

func GetRegions(c *gin.Context) {
	log.Printf("[INFO] GetRegions called")

	data, err := service.FetchRegions()
	if err != nil {
		log.Printf("[ERROR] FetchRegions failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var filtered []string
	supportedSet := make(map[string]struct{})

	for _, r := range utils.SupportedRegions {
		supportedSet[r] = struct{}{}
	}

	for _, r := range data {
		if _, ok := supportedSet[r]; ok {
			filtered = append(filtered, r)
		}
	}

	log.Printf("[INFO] Retrieved %d regions", len(filtered))
	c.JSON(http.StatusOK, filtered)
}

func GetTopVideos(c *gin.Context) {
	region := c.Query("region")
	category := c.Query("category")
	maxResultsStr := c.DefaultQuery("maxResults", "10")

	log.Printf("[INFO] GetTopVideos called with region: %s, category: %s, maxResults: %s", region, category, maxResultsStr)

	if region == "" || category == "" {
		log.Printf("[WARN] Missing region or category")
		c.JSON(http.StatusBadRequest, gin.H{"error": "region and category are required"})
		return
	}

	maxResults, err := strconv.Atoi(maxResultsStr)
	if err != nil || maxResults <= 0 || maxResults > 50 {
		log.Printf("[WARN] Invalid maxResults: %s", maxResultsStr)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid maxResults, must be between 1 and 50"})
		return
	}

	data, err := service.FetchTopVideos(region, category, maxResults)
	if err != nil {
		log.Printf("[ERROR] FetchTopVideos failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("[INFO] Retrieved %d top videos for region=%s, category=%s", len(data), region, category)
	c.JSON(http.StatusOK, data)
}

func GetAllTrending(c *gin.Context) {
	maxResultsStr := c.DefaultQuery("maxResults", "5")
	log.Printf("[INFO] GetAllTrending called with maxResults=%s", maxResultsStr)

	maxResults, err := strconv.Atoi(maxResultsStr)
	if err != nil || maxResults <= 0 {
		log.Printf("[WARN] Invalid maxResults: %s", maxResultsStr)
		c.JSON(http.StatusBadRequest, gin.H{"error": "maxResults must be a positive integer"})
		return
	}

	result := service.FetchAllTrending(maxResults)
	log.Printf("[INFO] GetAllTrending completed with results from %d regions", len(result))
	c.JSON(http.StatusOK, result)
}

func SearchVideos(c *gin.Context) {
	query := c.Query("query")
	region := c.DefaultQuery("region", "US")

	log.Printf("[INFO] SearchVideos called with query='%s', region='%s'", query, region)

	if query == "" {
		log.Printf("[WARN] Missing search query")
		c.JSON(http.StatusBadRequest, gin.H{"error": "query is required"})
		return
	}

	results, err := service.SearchVideosByQuery(query, region)
	if err != nil {
		log.Printf("[ERROR] SearchVideosByQuery failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("[INFO] Retrieved %d search results for query='%s'", len(results), query)
	c.JSON(http.StatusOK, results)
}
