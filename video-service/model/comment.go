package model

type CommentThreadResponse struct {
	Items []CommentThread `json:"items"`
	NextPageToken string   `json:"nextPageToken,omitempty"`
}

type CommentThread struct {
	ID      string        `json:"id"`
	Snippet struct {
		TopLevelComment Comment `json:"topLevelComment"`
		TotalReplyCount int     `json:"totalReplyCount"`
	} `json:"snippet"`
	Replies struct {
		Comments []Comment `json:"comments"`
	} `json:"replies,omitempty"`
}

type Comment struct {
	ID      string `json:"id"`
	Snippet struct {
		TextDisplay   string `json:"textDisplay"`
		AuthorDisplayName string `json:"authorDisplayName"`
		PublishedAt   string `json:"publishedAt"`
	} `json:"snippet"`
}
