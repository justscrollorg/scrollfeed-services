package config

import (
	"log"
	"os"
)

type Config struct {
	APIKey string
}

func Load() Config {
	key := os.Getenv("YOUTUBE_API_KEY")
	if key == "" {
		log.Fatal("YOUTUBE_API_KEY not set")
	}
	return Config{APIKey: key}
}
