package service

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"	
	"video-service/model"
)


func FetchVideoAndChannelStats(videoID string) (*model.VideoAndChannelStats, error) {
	log.Printf("[INFO] Fetching video and channel stats for videoID: %s", videoID)

	videoURL := fmt.Sprintf("https://www.googleapis.com/youtube/v3/videos?part=snippet,statistics&id=%s&key=%s", videoID, cfg.APIKey)
	log.Printf("[DEBUG] Video stats request URL: %s", videoURL)

	videoResp, err := http.Get(videoURL)
	if err != nil {
		log.Printf("[ERROR] Failed to fetch video stats: %v", err)
		return nil, err
	}
	defer videoResp.Body.Close()

	var videoData model.VideoStatsResponse
	if err := json.NewDecoder(videoResp.Body).Decode(&videoData); err != nil {
		log.Printf("[ERROR] Failed to decode video stats: %v", err)
		return nil, err
	}
	if len(videoData.Items) == 0 {
		log.Printf("[WARN] No video found for ID: %s", videoID)
		return nil, fmt.Errorf("no video found for ID %s", videoID)
	}

	videoStats := videoData.Items[0].Statistics
	channelID := videoData.Items[0].Snippet.ChannelId
	log.Printf("[INFO] Video fetched. ChannelID: %s", channelID)

	// Fetch channel stats
	channelURL := fmt.Sprintf("https://www.googleapis.com/youtube/v3/channels?part=statistics&id=%s&key=%s", channelID, cfg.APIKey)
	log.Printf("[DEBUG] Channel stats request URL: %s", channelURL)

	channelResp, err := http.Get(channelURL)
	if err != nil {
		log.Printf("[ERROR] Failed to fetch channel stats: %v", err)
		return nil, err
	}
	defer channelResp.Body.Close()

	var channelData model.ChannelStatsResponse
	if err := json.NewDecoder(channelResp.Body).Decode(&channelData); err != nil {
		log.Printf("[ERROR] Failed to decode channel stats: %v", err)
		return nil, err
	}
	if len(channelData.Items) == 0 {
		log.Printf("[WARN] No channel found for ID: %s", channelID)
		return nil, fmt.Errorf("no channel found for ID %s", channelID)
	}

	channelStats := channelData.Items[0].Statistics
	log.Printf("[INFO] Stats retrieved. VideoID: %s | Views: %s | Likes: %s | Comments: %s | Subscribers: %s",
		videoID, videoStats.ViewCount, videoStats.LikeCount, videoStats.CommentCount, channelStats.SubscriberCount)

	return &model.VideoAndChannelStats{
		ViewCount:       videoStats.ViewCount,
		LikeCount:       videoStats.LikeCount,
		CommentCount:    videoStats.CommentCount,
		SubscriberCount: channelStats.SubscriberCount,
	}, nil
}
