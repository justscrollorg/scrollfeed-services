package handler

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"news-service/model"
	"strings"
	"time"
)

// NewsStrategy interface for different fetching strategies
type NewsStrategy interface {
	FetchNews(region string, config *NewsConfig) ([]model.Article, error)
	GetName() string
}

// APIStrategy for GNews/NewsAPI
type APIStrategy struct{}

func (a *APIStrategy) GetName() string {
	return "API"
}

func (a *APIStrategy) FetchNews(region string, config *NewsConfig) ([]model.Article, error) {
	log.Printf("Fetching news via API strategy for region: %s", region)
	return fetchRegionNewsAPI(region, config)
}

// RSSStrategy for RSS feeds
type RSSStrategy struct {
	Sources map[string][]string
}

func NewRSSStrategy() *RSSStrategy {
	return &RSSStrategy{
		Sources: map[string][]string{
			"in": {
				"https://feeds.feedburner.com/ndtvnews-top-stories",
				"https://timesofindia.indiatimes.com/rssfeedstopstories.cms",
			},
		},
	}
}

func (r *RSSStrategy) GetName() string {
	return "RSS"
}

func (r *RSSStrategy) FetchNews(region string, config *NewsConfig) ([]model.Article, error) {
	log.Printf("Fetching news via RSS strategy for region: %s", region)

	sources, exists := r.Sources[region]
	if !exists {
		return nil, fmt.Errorf("no RSS sources configured for region: %s", region)
	}

	var allArticles []model.Article

	for _, source := range sources {
		articles, err := r.fetchFromRSSSource(source, region)
		if err != nil {
			log.Printf("Failed to fetch from RSS source %s: %v", source, err)
			continue
		}
		allArticles = append(allArticles, articles...)

		// Rate limiting between sources
		time.Sleep(config.RateLimit)
	}

	// Limit articles
	if len(allArticles) > config.MaxArticles {
		allArticles = allArticles[:config.MaxArticles]
	}

	log.Printf("Fetched %d articles via RSS for region=%s", len(allArticles), region)
	return allArticles, nil
}

// RSS Feed structures
type RSSFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Channel RSSChannel `xml:"channel"`
}

type RSSChannel struct {
	Title       string    `xml:"title"`
	Description string    `xml:"description"`
	Items       []RSSItem `xml:"item"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
}

func (r *RSSStrategy) fetchFromRSSSource(sourceURL, region string) ([]model.Article, error) {
	log.Printf("Fetching RSS from: %s", sourceURL)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(sourceURL)
	if err != nil {
		return nil, fmt.Errorf("HTTP error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-200 response: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body error: %v", err)
	}

	var feed RSSFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("XML parse error: %v", err)
	}

	var articles []model.Article
	for _, item := range feed.Channel.Items {
		// Parse publication date
		pubDate, _ := time.Parse(time.RFC1123Z, item.PubDate)
		if pubDate.IsZero() {
			pubDate, _ = time.Parse("Mon, 02 Jan 2006 15:04:05 -0700", item.PubDate)
		}
		if pubDate.IsZero() {
			pubDate = time.Now()
		}

		// Extract image from description or use fallback
		imageURL := extractImageFromHTML(item.Description)
		if imageURL == "" {
			imageURL = getDefaultNewsImage(region)
		}

		article := model.Article{
			Title:       strings.TrimSpace(item.Title),
			Description: strings.TrimSpace(stripHTML(item.Description)),
			URL:         strings.TrimSpace(item.Link),
			Image:       imageURL,
			Source: struct {
				Name string `json:"name" bson:"name"`
			}{
				Name: extractDomainName(sourceURL),
			},
			PublishedAt: pubDate,
			Topic:       region,
			FetchedAt:   time.Now(),
		}

		// Skip empty articles
		if article.Title != "" && article.URL != "" {
			articles = append(articles, article)
		}
	}

	log.Printf("Parsed %d articles from RSS source: %s", len(articles), sourceURL)
	return articles, nil
}

// Utility functions
func stripHTML(html string) string {
	// Simple HTML tag removal
	result := html
	for {
		start := strings.Index(result, "<")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], ">")
		if end == -1 {
			break
		}
		result = result[:start] + result[start+end+1:]
	}
	return strings.TrimSpace(result)
}

func extractImageFromHTML(html string) string {
	// Look for img tags in the HTML content
	start := strings.Index(strings.ToLower(html), "<img")
	if start == -1 {
		return ""
	}

	// Find the src attribute
	imgTag := html[start:]
	end := strings.Index(imgTag, ">")
	if end == -1 {
		return ""
	}
	imgTag = imgTag[:end+1]

	// Extract src value
	srcStart := strings.Index(strings.ToLower(imgTag), "src=")
	if srcStart == -1 {
		return ""
	}

	srcStart += 4 // Skip "src="
	if srcStart >= len(imgTag) {
		return ""
	}

	quote := imgTag[srcStart]
	if quote != '"' && quote != '\'' {
		return ""
	}

	srcStart++ // Skip opening quote
	srcEnd := strings.Index(imgTag[srcStart:], string(quote))
	if srcEnd == -1 {
		return ""
	}

	imageURL := imgTag[srcStart : srcStart+srcEnd]
	
	// Validate URL
	if strings.HasPrefix(imageURL, "http://") || strings.HasPrefix(imageURL, "https://") {
		return imageURL
	}
	
	return ""
}

func getDefaultNewsImage(region string) string {
	// Default news images by region
	defaultImages := map[string]string{
		"in": "https://images.unsplash.com/photo-1504711434969-e33886168f5c?w=400&h=200&fit=crop&crop=entropy&auto=format&q=80",
		"us": "https://images.unsplash.com/photo-1495020689067-958852a7765e?w=400&h=200&fit=crop&crop=entropy&auto=format&q=80",
		"de": "https://images.unsplash.com/photo-1586953208448-b95a79798f07?w=400&h=200&fit=crop&crop=entropy&auto=format&q=80",
		"gb": "https://images.unsplash.com/photo-1520637836862-4d197d17c155?w=400&h=200&fit=crop&crop=entropy&auto=format&q=80",
		"ca": "https://images.unsplash.com/photo-1506905925346-21bda4d32df4?w=400&h=200&fit=crop&crop=entropy&auto=format&q=80",
		"au": "https://images.unsplash.com/photo-1506197603052-3cc9c3a201bd?w=400&h=200&fit=crop&crop=entropy&auto=format&q=80",
	}

	if img, exists := defaultImages[region]; exists {
		return img
	}

	// Generic news fallback
	return "https://images.unsplash.com/photo-1504711434969-e33886168f5c?w=400&h=200&fit=crop&crop=entropy&auto=format&q=80"
}

func extractDomainName(url string) string {
	parts := strings.Split(url, "/")
	if len(parts) >= 3 {
		domain := parts[2]
		// Remove www. prefix
		domain = strings.TrimPrefix(domain, "www.")
		return domain
	}
	return "Unknown"
}

// Original API fetching function (refactored)
func fetchRegionNewsAPI(region string, config *NewsConfig) ([]model.Article, error) {
	var baseURL string

	if strings.Contains(config.BaseURL, "newsapi.org") {
		baseURL = fmt.Sprintf("%s?country=%s&pageSize=20&apiKey=%s",
			config.BaseURL, region, config.APIKey)
	} else {
		baseURL = fmt.Sprintf("%s?lang=en&country=%s&max=10&token=%s",
			config.BaseURL, region, config.APIKey)
	}

	var allArticles []model.Article

	for page := 1; page <= config.MaxPages; page++ {
		url := fmt.Sprintf("%s&page=%d", baseURL, page)

		log.Printf("Fetching API region=%s page=%d URL=%s", region, page, url)

		resp, err := http.Get(url)
		if err != nil {
			log.Printf("HTTP error for region=%s page=%d: %v", region, page, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("Non-200 response for region=%s page=%d: %s", region, page, resp.Status)
			continue
		}

		var result struct {
			Articles []model.Article `json:"articles"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			log.Printf("JSON decode error for region=%s page=%d: %v", region, page, err)
			continue
		}

		for _, article := range result.Articles {
			article.Topic = region
			article.FetchedAt = time.Now()
			allArticles = append(allArticles, article)
		}

		time.Sleep(config.RateLimit)
	}

	if len(allArticles) > config.MaxArticles {
		allArticles = allArticles[:config.MaxArticles]
	}

	log.Printf("Fetched %d articles via API for region=%s", len(allArticles), region)
	return allArticles, nil
}
