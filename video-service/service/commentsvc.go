package service

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"video-service/config"
	"video-service/model"
)

func FetchAllComments(videoID string) ([]model.Comment, error) {
	log.Printf("[INFO] Fetching all comments for videoID: %s", videoID)

	var allComments []model.Comment
	apiKey := config.Load().APIKey
	baseURL := "https://www.googleapis.com/youtube/v3/commentThreads"
	pageToken := ""
	pageCount := 0

	for {
		apiURL := fmt.Sprintf(
			"%s?part=snippet,replies&videoId=%s&maxResults=100&key=%s&pageToken=%s",
			baseURL, videoID, apiKey, pageToken,
		)

		log.Printf("[DEBUG] Request URL: %s", apiURL)

		resp, err := http.Get(apiURL)
		if err != nil {
			log.Printf("[ERROR] Failed to fetch comments: %v", err)
			return nil, err
		}
		defer resp.Body.Close()

		var result model.CommentThreadResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			log.Printf("[ERROR] Failed to decode comment response: %v", err)
			return nil, err
		}

		log.Printf("[DEBUG] Page %d: %d top-level comments received", pageCount+1, len(result.Items))

		for _, item := range result.Items {
			allComments = append(allComments, item.Snippet.TopLevelComment)
			allComments = append(allComments, item.Replies.Comments...)
		}

		if result.NextPageToken == "" {
			break
		}
		pageToken = result.NextPageToken
		pageCount++
	}

	log.Printf("[INFO] Total comments (including replies) fetched: %d for videoID: %s", len(allComments), videoID)
	return allComments, nil
}
