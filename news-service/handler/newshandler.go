package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"news-service/model"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var regions = []string{"us", "in", "de"} // GNews supported countries

func fetchRegionNews(region, token string) ([]model.Article, error) {
	baseURL := fmt.Sprintf("https://gnews.io/api/v4/top-headlines?lang=en&country=%s&max=10&token=%s", region, token)
	var allArticles []model.Article

	for page := 1; page <= 4; page++ {
		url := fmt.Sprintf("%s&page=%d", baseURL, page)
		log.Printf("Fetching region=%s page=%d URL=%s", region, page, url)

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

		time.Sleep(1 * time.Second) // rate limit
	}

	if len(allArticles) > 33 {
		allArticles = allArticles[:33]
	}
	log.Printf("Fetched %d articles for region=%s", len(allArticles), region)
	return allArticles, nil
}

func StartScheduledFetcher(db *mongo.Database) {
	token := os.Getenv("GNEWS_API_KEY")
	if token == "" {
		log.Fatal("Missing GNEWS_API_KEY")
	}

	log.Println("Starting scheduled news fetcher...")

	// run immediately
	fetchAndStoreArticles(db, token)

	ticker := time.NewTicker(8 * time.Hour)
	for {
		<-ticker.C
		fetchAndStoreArticles(db, token)
	}
}

func fetchAndStoreArticles(db *mongo.Database, token string) {
	log.Println("Fetching GNews articles by region...")

	for _, region := range regions {
		log.Printf("Region: %s", region)

		articles, err := fetchRegionNews(region, token)
		if err != nil {
			log.Printf("Failed to fetch news for region=%s: %v", region, err)
			continue
		}

		log.Printf("Inserting %d articles for region=%s", len(articles), region)

		for i, article := range articles {
			filter := bson.M{"url": article.URL}
			update := bson.M{"$set": article}
			_, err := db.Collection("articles").UpdateOne(context.TODO(), filter, update, options.Update().SetUpsert(true))
			if err != nil {
				log.Printf("[%d] Insert failed for article: %s | error: %v", i, article.URL, err)
			} else {
				log.Printf("[%d] Upserted article: %s", i, article.URL)
			}
		}
	}

	log.Println("Finished fetch and store cycle")
}
