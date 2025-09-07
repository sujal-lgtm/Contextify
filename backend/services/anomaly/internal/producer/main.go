package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"
)

type Event struct {
	TraceID   string `json:"trace_id"`
	Service   string `json:"service"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

func main() {
	// Get Kafka configuration from environment
	brokers := os.Getenv("KAFKA_BROKERS")
	if brokers == "" {
		brokers = "localhost:9092"
	}

	topic := os.Getenv("KAFKA_TOPIC")
	if topic == "" {
		topic = "contextify-events"
	}

	// Create Kafka writer
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:  []string{brokers},
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	})
	defer writer.Close()

	// Graceful shutdown
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	log.Printf("🚀 Anomaly producer starting, publishing to topic: %s", topic)

	// Simulate producing events
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			event := Event{
				TraceID:   "test-trace-" + time.Now().Format("20060102150405"),
				Service:   "anomaly-service",
				Level:     "INFO",
				Message:   "Test event from anomaly producer",
				Timestamp: time.Now().Format(time.RFC3339),
			}

			bytes, err := json.Marshal(event)
			if err != nil {
				log.Printf("Failed to marshal event: %v", err)
				continue
			}

			err = writer.WriteMessages(context.Background(), kafka.Message{
				Key:   []byte(event.TraceID),
				Value: bytes,
			})
			if err != nil {
				log.Printf("Failed to publish event: %v", err)
			} else {
				log.Printf("�� Published event: %s", event.TraceID)
			}

		case <-signals:
			log.Println("🛑 Anomaly producer shutting down...")
			return
		}
	}
}
