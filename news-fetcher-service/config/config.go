package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

type Config struct {
	MongoURI      string
	NATSUrl       string
	GNewsAPIKey   string
	FetchInterval time.Duration
	RateLimit     time.Duration
	MaxRetries    int
	RetryDelay    time.Duration
	WorkerCount   int
}

func Load() *Config {
	cfg := &Config{
		MongoURI:      getEnv("MONGO_URI", "mongodb://localhost:27017"),
		NATSUrl:       getEnv("NATS_URL", "nats://localhost:4222"),
		GNewsAPIKey:   getEnv("GNEWS_API_KEY", ""),
		FetchInterval: getDurationEnv("FETCH_INTERVAL", "4h"),
		RateLimit:     getDurationEnv("RATE_LIMIT", "1s"),
		MaxRetries:    getIntEnv("MAX_RETRIES", 3),
		RetryDelay:    getDurationEnv("RETRY_DELAY", "30s"),
		WorkerCount:   getIntEnv("WORKER_COUNT", 3),
	}

	if cfg.GNewsAPIKey == "" {
		log.Fatal("GNEWS_API_KEY is required")
	}

	log.Printf("Config loaded - FetchInterval: %v, RateLimit: %v, Workers: %d",
		cfg.FetchInterval, cfg.RateLimit, cfg.WorkerCount)

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue string) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	duration, _ := time.ParseDuration(defaultValue)
	return duration
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
