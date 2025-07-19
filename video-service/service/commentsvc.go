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
	cfg := config.Load()
	baseURL := "https://www.googleapis.com/youtube/v3/commentThreads"
	pageToken := ""
	pageCount := 0

	for {
		apiURL := fmt.Sprintf(
			"%s?part=snippet,replies&videoId=%s&maxResults=100&key=%s&pageToken=%s",
			baseURL, videoID, cfg.YouTubeAPIKey, pageToken,
		)

		log.Printf("[DEBUG] Request URL: %s", apiURL)

		resp, err := http.Get(apiURL)
		if err != nil {
			log.Printf("[ERROR] Failed to fetch comments: %v", err)
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("[ERROR] YouTube API returned status: %d", resp.StatusCode)
			return nil, fmt.Errorf("YouTube API error: %d", resp.StatusCode)
		}

		var response struct {
			Items []struct {
				Snippet struct {
					TopLevelComment struct {
						Snippet model.Comment `json:"snippet"`
					} `json:"topLevelComment"`
				} `json:"snippet"`
				Replies struct {
					Comments []struct {
						Snippet model.Comment `json:"snippet"`
					} `json:"comments"`
				} `json:"replies"`
			} `json:"items"`
			NextPageToken string `json:"nextPageToken"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			log.Printf("[ERROR] Failed to decode comments response: %v", err)
			return nil, err
		}

		for _, item := range response.Items {
			allComments = append(allComments, item.Snippet.TopLevelComment.Snippet)

			// Add replies if they exist
			for _, reply := range item.Replies.Comments {
				allComments = append(allComments, reply.Snippet)
			}
		}

		pageCount++
		pageToken = response.NextPageToken

		if pageToken == "" || pageCount >= 5 { // Limit to 5 pages to avoid excessive requests
			break
		}
	}

	log.Printf("[INFO] Successfully fetched %d comments for video: %s", len(allComments), videoID)
	return allComments, nil
}
