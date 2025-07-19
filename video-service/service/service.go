package service

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"video-service/config"
	"video-service/model"
)

var cfg = config.Load()

func FetchCategories(region string) ([]model.YouTubeCategory, error) {
	apiURL := fmt.Sprintf("https://www.googleapis.com/youtube/v3/videoCategories?part=snippet&regionCode=%s&key=%s", region, cfg.APIKey)
	log.Printf("[INFO] Fetching categories for region: %s", region)
	log.Printf("[DEBUG] Request URL: %s", apiURL)

	resp, err := http.Get(apiURL)
	if err != nil {
		log.Printf("[ERROR] Failed to fetch categories: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Printf("[ERROR] YouTube API returned status: %d", resp.StatusCode)
		log.Printf("[ERROR] Response body: %s", string(bodyBytes))
		return nil, fmt.Errorf("YouTube API error: %s", resp.Status)
	}

	var result model.CategoryListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("[ERROR] Failed to decode category response: %v", err)
		return nil, err
	}

	var categories []model.YouTubeCategory
	for _, item := range result.Items {
		if item.Snippet.Assignable {
			categories = append(categories, model.YouTubeCategory{
				ID:    item.ID,
				Title: item.Snippet.Title,
			})
		}
	}

	log.Printf("[INFO] %d categories fetched for region: %s", len(categories), region)
	return categories, nil
}

func FetchRegions() ([]string, error) {
	apiURL := fmt.Sprintf("https://www.googleapis.com/youtube/v3/i18nRegions?part=snippet&key=%s", cfg.APIKey)
	log.Printf("[INFO] Fetching regions")
	log.Printf("[DEBUG] Request URL: %s", apiURL)

	resp, err := http.Get(apiURL)
	if err != nil {
		log.Printf("[ERROR] Failed to fetch regions: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	var result model.RegionListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("[ERROR] Failed to decode region response: %v", err)
		return nil, err
	}

	var regions []string
	for _, item := range result.Items {
		regions = append(regions, item.ID)
	}
	log.Printf("[INFO] %d regions fetched", len(regions))
	return regions, nil
}

func FetchTopVideos(region string, category string, maxResults int) ([]model.VideoItem, error) {
	apiURL := fmt.Sprintf(
		"https://www.googleapis.com/youtube/v3/videos?part=snippet&chart=mostPopular&regionCode=%s&videoCategoryId=%s&maxResults=%d&key=%s",
		region, category, maxResults, cfg.APIKey,
	)
	log.Printf("[INFO] Fetching top videos for region=%s, category=%s, maxResults=%d", region, category, maxResults)
	log.Printf("[DEBUG] Request URL: %s", apiURL)

	resp, err := http.Get(apiURL)
	if err != nil {
		log.Printf("[ERROR] Failed to fetch top videos: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	var result model.YouTubeVideoResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("[ERROR] Failed to decode top videos response: %v", err)
		return nil, err
	}
	for i := range result.Items {
		result.Items[i].VideoURL = fmt.Sprintf("https://www.youtube.com/watch?v=%s", result.Items[i].ID)
	}
	log.Printf("[INFO] %d top videos fetched for region=%s, category=%s", len(result.Items), region, category)
	return result.Items, nil
}

func FetchAllTrending(maxResults int) map[string]map[string][]model.VideoItem {
	log.Printf("[INFO] Fetching all trending videos with maxResults=%d", maxResults)
	data := make(map[string]map[string][]model.VideoItem)

	regions, err := FetchRegions()
	if err != nil {
		log.Printf("[ERROR] Failed to fetch regions: %v", err)
		return data
	}

	for _, region := range regions {
		categories, err := FetchCategories(region)
		if err != nil {
			log.Printf("[ERROR] Failed to fetch categories for region %s: %v", region, err)
			continue
		}

		data[region] = make(map[string][]model.VideoItem)
		for _, cat := range categories {
			videos, err := FetchTopVideos(region, cat.ID, maxResults)
			if err != nil {
				log.Printf("[ERROR] Failed to fetch videos for region=%s, categoryID=%s: %v", region, cat.ID, err)
				continue
			}
			data[region][cat.Title] = videos
		}
		log.Printf("[INFO] Fetched trending videos for region=%s", region)
	}

	return data
}

func SearchVideosByQuery(query, region string) ([]model.SearchVideoItem, error) {
	encodedQuery := url.QueryEscape(query)
	apiURL := fmt.Sprintf(
		"https://www.googleapis.com/youtube/v3/search?part=snippet&maxResults=10&type=video&regionCode=%s&q=%s&key=%s",
		region, encodedQuery, cfg.APIKey)

	log.Printf("[INFO] Searching videos for query='%s' in region=%s", query, region)
	log.Printf("[DEBUG] Request URL: %s", apiURL)

	resp, err := http.Get(apiURL)
	if err != nil {
		log.Printf("[ERROR] Failed to search videos: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	var result model.SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("[ERROR] Failed to decode search response: %v", err)
		return nil, err
	}

	for i := range result.Items {
		result.Items[i].VideoURL = fmt.Sprintf("https://www.youtube.com/watch?v=%s", result.Items[i].ID.VideoID)
	}
	log.Printf("[INFO] %d videos found for query='%s'", len(result.Items), query)

	return result.Items, nil
}
