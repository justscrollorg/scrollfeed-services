package model

type VideoItem struct {
	ID       string `json:"id"`
	VideoURL string `json:"videoUrl,omitempty"`
	Snippet  struct {
		Title        string     `json:"title"`
		ChannelTitle string     `json:"channelTitle"`
		PublishedAt  string     `json:"publishedAt"`
		Thumbnails   Thumbnails `json:"thumbnails"`
	} `json:"snippet"`
}

type YouTubeVideoResponse struct {
	Items []VideoItem `json:"items"`
}

type YouTubeCategory struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type CategoryListResponse struct {
	Items []struct {
		ID      string `json:"id"`
		Snippet struct {
			Title      string `json:"title"`
			Assignable bool   `json:"assignable"`
		} `json:"snippet"`
	} `json:"items"`
}

type RegionListResponse struct {
	Items []struct {
		ID      string `json:"id"`
		Snippet struct {
			Name string `json:"name"`
		} `json:"snippet"`
	} `json:"items"`
}

type SearchVideoItem struct {
	ID struct {
		VideoID string `json:"videoId"`
	} `json:"id"`
	Snippet struct {
		Title        string     `json:"title"`
		ChannelTitle string     `json:"channelTitle"`
		PublishedAt  string     `json:"publishedAt"`
		Thumbnails   Thumbnails `json:"thumbnails"`
	} `json:"snippet"`
	VideoURL string `json:"videoUrl,omitempty"`
}

type SearchResponse struct {
	Items []SearchVideoItem `json:"items"`
}

type Thumbnail struct {
	URL    string `json:"url"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

type Thumbnails struct {
	Default  Thumbnail `json:"default"`
	Medium   Thumbnail `json:"medium"`
	High     Thumbnail `json:"high"`
	Standard Thumbnail `json:"standard"`
	Maxres   Thumbnail `json:"maxres"`
}
