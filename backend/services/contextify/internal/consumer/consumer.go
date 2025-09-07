package consumer

import (
	"encoding/json"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Shopify/sarama"
	"github.com/sirupsen/logrus"
	"github.com/sujal-lgtm/Contextify/backend/services/contextify/internal/db"
)

// Event represents a Kafka event message
type Event struct {
	TraceID   string `json:"trace_id"`
	Service   string `json:"service"`
	Level     string `json:"level"`      // maps to Status in DB
	Message   string `json:"message"`    // optional
	Timestamp string `json:"timestamp"`  // ISO8601 string
	LatencyMs int    `json:"latency_ms"` // optional
	QueueLen  int    `json:"queue_length"`
}

// Start consumes Kafka messages and saves them to DB
func Start(brokers string, topic string, database *db.DB) error {
	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true
	config.Version = sarama.V2_1_0_0

	consumer, err := sarama.NewConsumer(strings.Split(brokers, ","), config)
	if err != nil {
		return err
	}
	defer consumer.Close()

	partitionConsumer, err := consumer.ConsumePartition(topic, 0, sarama.OffsetNewest)
	if err != nil {
		return err
	}
	defer partitionConsumer.Close()

	// Graceful shutdown channel
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	logrus.Infof("✅ Listening for Kafka messages on topic: %s", topic)

	for {
		select {
		case msg := <-partitionConsumer.Messages():
			var e Event
			if err := json.Unmarshal(msg.Value, &e); err != nil {
				logrus.Errorf("Failed to unmarshal Kafka message: %v", err)
				continue
			}

			// Parse timestamp
			t, err := time.Parse(time.RFC3339, e.Timestamp)
			var ts int64
			if err != nil {
				logrus.Warnf("Failed to parse timestamp, using current time: %v", err)
				ts = time.Now().UnixMilli()
			} else {
				ts = t.UnixMilli()
			}

			// Map Kafka event to DB Event
			dbEvent := db.Event{
				Service:     e.Service,
				Timestamp:   ts,
				Status:      e.Level,
				LatencyMs:   e.LatencyMs,
				QueueLength: e.QueueLen,
			}

			// Save to DB
			if err := database.SaveContext(dbEvent); err != nil {
				logrus.Errorf("Failed to save event to DB: %v", err)
				continue
			}

			logrus.WithFields(logrus.Fields{
				"trace_id": e.TraceID,
				"service":  e.Service,
				"status":   e.Level,
				"latency":  e.LatencyMs,
				"queue":    e.QueueLen,
			}).Info("📥 Event saved to DB")

		case err := <-partitionConsumer.Errors():
			logrus.Errorf("Kafka consumer error: %v", err)

		case <-signals:
			logrus.Info("🛑 Kafka consumer shutting down...")
			return nil
		}
	}
}
