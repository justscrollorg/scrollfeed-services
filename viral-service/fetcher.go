package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// FetchRedditViral fetches viral stories from Reddit
func FetchRedditViral() ([]ViralStory, error) {
	subreddits := []string{"worldnews", "news", "technology"}
	var allStories []ViralStory

	client := &http.Client{Timeout: 10 * time.Second}

	for _, subreddit := range subreddits {
		url := fmt.Sprintf("https://www.reddit.com/r/%s/hot.json?limit=25", subreddit)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			log.Printf("Error creating request for r/%s: %v", subreddit, err)
			continue
		}
		req.Header.Set("User-Agent", "JustScrolls-Viral-Fetcher/1.0")

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Error fetching from r/%s: %v", subreddit, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			log.Printf("Reddit API returned status %d for r/%s", resp.StatusCode, subreddit)
			continue
		}

		var redditResp RedditResponse
		if err := json.NewDecoder(resp.Body).Decode(&redditResp); err != nil {
			log.Printf("Error decoding Reddit response for r/%s: %v", subreddit, err)
			continue
		}

		for _, post := range redditResp.Data.Children {
			// Filter out low-engagement posts
			if post.Data.Score < 100 {
				continue
			}

			// Calculate viral score
			viralScore := calculateViralScore(post.Data.Score, post.Data.NumComments, 0)

			imageURL := post.Data.Thumbnail
			if imageURL == "self" || imageURL == "default" || imageURL == "nsfw" {
				imageURL = ""
			}

			story := ViralStory{
				ID:          "reddit_" + post.Data.ID,
				Title:       post.Data.Title,
				Description: truncateText(post.Data.Selftext, 300),
				URL:         "https://reddit.com" + post.Data.Permalink,
				ImageURL:    imageURL,
				Source:      "reddit",
				SourceID:    post.Data.ID,
				Author:      post.Data.Author,
				Category:    post.Data.Subreddit,
				PublishedAt: time.Unix(int64(post.Data.Created), 0),
				FetchedAt:   time.Now(),
				ViralScore:  viralScore,
				Upvotes:     post.Data.Score,
				Comments:    post.Data.NumComments,
				Shares:      0,
				Engagement:  float64(post.Data.Score+post.Data.NumComments) / 100.0,
			}

			allStories = append(allStories, story)
		}

		log.Printf("Fetched %d viral stories from r/%s", len(redditResp.Data.Children), subreddit)

		// Rate limiting - be respectful to Reddit API
		time.Sleep(2 * time.Second)
	}

	return allStories, nil
}

// FetchHackerNewsViral fetches top stories from HackerNews
func FetchHackerNewsViral() ([]ViralStory, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	var allStories []ViralStory

	// Get top story IDs
	resp, err := client.Get("https://hacker-news.firebaseio.com/v0/topstories.json")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch HN top stories: %w", err)
	}
	defer resp.Body.Close()

	var storyIDs []int
	if err := json.NewDecoder(resp.Body).Decode(&storyIDs); err != nil {
		return nil, fmt.Errorf("failed to decode HN story IDs: %w", err)
	}

	// Fetch details for top 20 stories
	limit := 20
	if len(storyIDs) < limit {
		limit = len(storyIDs)
	}

	for i := 0; i < limit; i++ {
		storyID := storyIDs[i]

		itemURL := fmt.Sprintf("https://hacker-news.firebaseio.com/v0/item/%d.json", storyID)
		resp, err := client.Get(itemURL)
		if err != nil {
			log.Printf("Error fetching HN item %d: %v", storyID, err)
			continue
		}

		var item HackerNewsItem
		if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
			log.Printf("Error decoding HN item %d: %v", storyID, err)
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		// Skip jobs, polls, etc - only want stories
		if item.Type != "story" || item.URL == "" {
			continue
		}

		// Calculate viral score
		viralScore := calculateViralScore(item.Score, item.Descendants, 0)

		story := ViralStory{
			ID:          "hn_" + strconv.Itoa(item.ID),
			Title:       item.Title,
			Description: "Top story on Hacker News",
			URL:         item.URL,
			ImageURL:    "",
			Source:      "hackernews",
			SourceID:    strconv.Itoa(item.ID),
			Author:      item.By,
			Category:    "technology",
			PublishedAt: time.Unix(item.Time, 0),
			FetchedAt:   time.Now(),
			ViralScore:  viralScore,
			Upvotes:     item.Score,
			Comments:    item.Descendants,
			Shares:      0,
			Engagement:  float64(item.Score+item.Descendants) / 50.0,
		}

		allStories = append(allStories, story)

		// Rate limiting
		time.Sleep(100 * time.Millisecond)
	}

	log.Printf("Fetched %d viral stories from HackerNews", len(allStories))
	return allStories, nil
}

// calculateViralScore computes a viral score based on engagement metrics
func calculateViralScore(upvotes, comments, shares int) int {
	// Weighted formula: upvotes are worth more, but comments show engagement
	score := (upvotes * 3) + (comments * 5) + (shares * 2)

	// Normalize to 0-100 scale
	normalized := score / 100
	if normalized > 100 {
		normalized = 100
	}

	return normalized
}

// truncateText truncates text to maxLen characters
func truncateText(text string, maxLen int) string {
	text = strings.TrimSpace(text)
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}

// deduplicateStories removes duplicate stories based on similar titles
func deduplicateStories(stories []ViralStory) []ViralStory {
	seen := make(map[string]bool)
	var unique []ViralStory

	for _, story := range stories {
		// Create a normalized key from title
		key := strings.ToLower(strings.TrimSpace(story.Title))

		if !seen[key] {
			seen[key] = true
			unique = append(unique, story)
		}
	}

	return unique
}
