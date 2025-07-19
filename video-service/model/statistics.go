package model

type VideoStatsResponse struct {
	Items []struct {
		ID         string `json:"id"`
		Statistics struct {
			ViewCount    string `json:"viewCount"`
			LikeCount    string `json:"likeCount"`
			CommentCount string `json:"commentCount"`
		} `json:"statistics"`
		Snippet struct {
			ChannelId string `json:"channelId"`
		} `json:"snippet"`
	} `json:"items"`
}

type ChannelStatsResponse struct {
	Items []struct {
		Statistics struct {
			SubscriberCount string `json:"subscriberCount"`
		} `json:"statistics"`
	} `json:"items"`
}

type VideoAndChannelStats struct {
	ViewCount       string `json:"viewCount"`
	LikeCount       string `json:"likeCount"`
	CommentCount    string `json:"commentCount"`
	SubscriberCount string `json:"subscriberCount"`
}
