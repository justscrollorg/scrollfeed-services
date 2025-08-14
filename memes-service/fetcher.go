package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
)

type ImgflipResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Memes []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"memes"`
	} `json:"data"`
}

func FetchImgflipMemes() ([]Meme, error) {
	resp, err := http.Get("https://api.imgflip.com/get_memes")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	var result ImgflipResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	memes := []Meme{}
	for _, m := range result.Data.Memes {
		memes = append(memes, Meme{
			Title:     m.Name,
			ImageURL:  m.URL,
			Source:    "imgflip",
			Permalink: m.URL,
		})
	}
	return memes, nil
}

// Reddit fetcher (top memes from r/memes)
type RedditListing struct {
	Data struct {
		Children []struct {
			Data struct {
				Title     string `json:"title"`
				URL       string `json:"url"`
				Permalink string `json:"permalink"`
			} `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

func FetchRedditMemes() ([]Meme, error) {
	url := "https://www.reddit.com/r/memes/top.json?limit=20&t=day"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "scrollfeed-memes-bot/0.1")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	var listing RedditListing
	if err := json.Unmarshal(body, &listing); err != nil {
		return nil, err
	}
	memes := []Meme{}
	for _, child := range listing.Data.Children {
		m := child.Data
		if m.URL != "" && (endsWith(m.URL, ".jpg") || endsWith(m.URL, ".png")) {
			memes = append(memes, Meme{
				Title:     m.Title,
				ImageURL:  m.URL,
				Source:    "reddit",
				Permalink: "https://reddit.com" + m.Permalink,
			})
		}
	}
	return memes, nil
}

func endsWith(s, suffix string) bool {
	if len(s) < len(suffix) {
		return false
	}
	return s[len(s)-len(suffix):] == suffix
}
