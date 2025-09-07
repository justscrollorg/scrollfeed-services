package main

import (
	"encoding/json"
	"io"
	"log"
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
	log.Println("Fetching memes from Imgflip...")
	resp, err := http.Get("https://api.imgflip.com/get_memes")
	if err != nil {
		log.Printf("Imgflip fetch error: %v", err)
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var result ImgflipResponse
	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("Imgflip unmarshal error: %v", err)
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
	log.Printf("Fetched %d memes from Imgflip", len(memes))
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
	log.Println("Fetching memes from Reddit...")
	url := "https://www.reddit.com/r/memes/top.json?limit=20&t=day"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; ScrollfeedBot/1.0)")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Reddit fetch error: %v", err)
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	
	// Debug: Log response status and first 200 chars of body
	log.Printf("Reddit response status: %d", resp.StatusCode)
	if len(body) > 200 {
		log.Printf("Reddit response body preview: %s...", string(body[:200]))
	} else {
		log.Printf("Reddit response body: %s", string(body))
	}
	
	var listing RedditListing
	if err := json.Unmarshal(body, &listing); err != nil {
		log.Printf("Reddit unmarshal error: %v", err)
		log.Printf("Response body was: %s", string(body))
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
	log.Printf("Fetched %d memes from Reddit", len(memes))
	return memes, nil
}

func endsWith(s, suffix string) bool {
	if len(s) < len(suffix) {
		return false
	}
	return s[len(s)-len(suffix):] == suffix
}
