package main

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

type Event struct {
	Service     string `json:"service"`
	Timestamp   int64  `json:"timestamp"`
	LatencyMs   int    `json:"latency_ms"`
	Status      string `json:"status"`
	QueueLength int    `json:"queue_length"`
}

func main() {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.Info("🚀 Anomaly producer starting...")

	w := kafka.NewWriter(kafka.WriterConfig{
		Brokers: []string{"kafka:9092"},
		Topic:   "contextify-events",
	})
	defer w.Close()

	// Wait for Kafka to be ready
	time.Sleep(5 * time.Second)

	// Create a channel to handle graceful shutdown
	done := make(chan bool)

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		logrus.Info("🛑 Shutting down producer...")
		done <- true
	}()

	// Produce events
	go func() {
		eventCounter := 0
		for {
			select {
			case <-done:
				return
			default:
				now := time.Now().UnixMilli()

				// Generate realistic events with some anomalies
				events := generateEvents(now, eventCounter)

				for _, e := range events {
					select {
					case <-done:
						return
					default:
						b, _ := json.Marshal(e)
						if err := w.WriteMessages(context.Background(), kafka.Message{Value: b}); err != nil {
							logrus.Errorf("Failed to write message: %v", err)
						} else {
							logrus.Infof("✅ Produced event %d: %s", eventCounter, string(b))
							eventCounter++
						}
						time.Sleep(250 * time.Millisecond)
					}
				}

				time.Sleep(5 * time.Second) // Wait before next batch
			}
		}
	}()

	<-done
	logrus.Info("✅ Producer stopped gracefully")
}

func generateEvents(baseTime int64, counter int) []Event {
	events := []Event{}

	// Normal events
	events = append(events, Event{
		Service:     "contextify",
		Timestamp:   baseTime + int64(counter*1000),
		LatencyMs:   120,
		Status:      "success",
		QueueLength: 15,
	})

	// Latency spike
	events = append(events, Event{
		Service:     "contextify",
		Timestamp:   baseTime + int64((counter+1)*1000),
		LatencyMs:   450,
		Status:      "success",
		QueueLength: 20,
	})

	// Error events
	events = append(events, Event{
		Service:     "contextify",
		Timestamp:   baseTime + int64((counter+2)*1000),
		LatencyMs:   130,
		Status:      "error",
		QueueLength: 25,
	})

	// Queue length spike
	events = append(events, Event{
		Service:     "contextify",
		Timestamp:   baseTime + int64((counter+3)*1000),
		LatencyMs:   110,
		Status:      "success",
		QueueLength: 150,
	})

	return events
}
