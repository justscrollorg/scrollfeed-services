// model/article.go
package model

import "time"

type Article struct {
	Title       string `json:"title" bson:"title"`
	Description string `json:"description" bson:"description"`
	URL         string `json:"url" bson:"url"`
	Image       string `json:"image" bson:"image"`
	Source      struct {
		Name string `json:"name" bson:"name"`
	} `json:"source" bson:"source"`
	PublishedAt time.Time `json:"publishedAt" bson:"publishedAt"`
	Topic       string    `json:"topic" bson:"topic"`
	FetchedAt   time.Time `json:"fetchedAt" bson:"fetchedAt"`
}
