package handler

import (
	"encoding/json"
	"log"
	"news-service/model"
	"time"

	"github.com/nats-io/nats.go"
)

// NATSConfig holds NATS configuration
type NATSConfig struct {
	URL     string
	Subject string
}

// NATSPublisher handles publishing news to NATS
type NATSPublisher struct {
	conn    *nats.Conn
	subject string
}

// NewNATSPublisher creates a new NATS publisher
func NewNATSPublisher(config *NATSConfig) (*NATSPublisher, error) {
	nc, err := nats.Connect(config.URL)
	if err != nil {
		return nil, err
	}

	return &NATSPublisher{
		conn:    nc,
		subject: config.Subject,
	}, nil
}

// Close closes the NATS connection
func (np *NATSPublisher) Close() {
	if np.conn != nil {
		np.conn.Close()
	}
}

// PublishNews publishes a news article to NATS
func (np *NATSPublisher) PublishNews(article model.Article) error {
	// Add metadata
	message := NewsMessage{
		Article:   article,
		Timestamp: time.Now(),
		Source:    "news-service",
		Version:   "1.0",
	}

	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	err = np.conn.Publish(np.subject, data)
	if err != nil {
		return err
	}

	log.Printf("Published news article to NATS: %s", article.Title)
	return nil
}

// PublishBatch publishes multiple articles in a batch
func (np *NATSPublisher) PublishBatch(articles []model.Article) error {
	for _, article := range articles {
		if err := np.PublishNews(article); err != nil {
			log.Printf("Failed to publish article to NATS: %v", err)
			continue
		}

		// Small delay between messages to avoid overwhelming
		time.Sleep(10 * time.Millisecond)
	}

	log.Printf("Published %d articles to NATS", len(articles))
	return nil
}

// NewsMessage represents the structure sent to NATS
type NewsMessage struct {
	Article   model.Article `json:"article"`
	Timestamp time.Time     `json:"timestamp"`
	Source    string        `json:"source"`
	Version   string        `json:"version"`
}
