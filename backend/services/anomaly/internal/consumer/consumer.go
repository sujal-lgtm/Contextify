package consumer

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/sujal-lgtm/Contextify/backend/services/anomaly/internal/detector"

	"github.com/segmentio/kafka-go"
)

func Start() error {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{"kafka:9092"},
		Topic:   "contextify-events",
		GroupID: "anomaly-service",
	})
	defer r.Close()

	w := kafka.NewWriter(kafka.WriterConfig{
		Brokers: []string{"kafka:9092"},
		Topic:   "anomalies",
	})
	defer w.Close()

	logrus.Info("📡 Listening for events on contextify-events...")

	// Initialize error rate tracker with 10-second window
	tracker := detector.NewErrorRateTracker(10)

	for {
		m, err := r.ReadMessage(context.Background())
		if err != nil {
			logrus.Errorf("❌ failed to read message: %v", err)
			continue
		}

		event, err := detector.ParseEvent(m.Value)
		if err != nil {
			logrus.Errorf("❌ failed to parse event: %v", err)
			continue
		}

		logrus.Infof(" Received event: %+v", event)

		// Add event to error rate tracker
		tracker.AddEvent(event)

		anomaliesDetected := false

		// 1️⃣ Latency spike
		if detector.CheckLatency(event, 300) {
			msg := detector.CreateAnomalyMessage(event, "latency_spike")
			err := w.WriteMessages(context.Background(), kafka.Message{Value: []byte(msg)})
			if err != nil {
				logrus.Errorf("Failed to write latency anomaly: %v", err)
			} else {
				logrus.Warnf("⚠️ Latency spike detected: %s", msg)
				detector.IncrementAnomalyCount()
				anomaliesDetected = true
			}
		}

		// 2️⃣ Error rate spike
		if tracker.CheckErrorRate(0.5) { // threshold: 50% errors in window
			msg := detector.CreateAnomalyMessage(event, "error_rate_spike")
			err := w.WriteMessages(context.Background(), kafka.Message{Value: []byte(msg)})
			if err != nil {
				logrus.Errorf("Failed to write error rate anomaly: %v", err)
			} else {
				logrus.Warnf("⚠️ Error rate spike detected: %s", msg)
				detector.IncrementAnomalyCount()
				anomaliesDetected = true
			}
		}

		// 3️⃣ Queue length threshold
		if detector.CheckQueueLength(event, 100) {
			msg := detector.CreateAnomalyMessage(event, "queue_length_spike")
			err := w.WriteMessages(context.Background(), kafka.Message{Value: []byte(msg)})
			if err != nil {
				logrus.Errorf("Failed to write queue length anomaly: %v", err)
			} else {
				logrus.Warnf("⚠️ Queue length spike detected: %s", msg)
				detector.IncrementAnomalyCount()
				anomaliesDetected = true
			}
		}

		if anomaliesDetected {
			logrus.Warnf("🚨 Anomaly detected for service: %s", event.Service)
		}
	}
}
