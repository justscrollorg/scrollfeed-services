// model/article.go
package model

import "time"

type Article struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	URL         string `json:"url"`
	Image       string `json:"image"`
	Source      struct {
		Name string `json:"name"`
	} `json:"source"`
	PublishedAt time.Time `json:"publishedAt"`
	Topic       string    `json:"topic"`
	FetchedAt   time.Time `json:"fetchedAt"`
}
