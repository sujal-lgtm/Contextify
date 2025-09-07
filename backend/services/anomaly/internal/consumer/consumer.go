package consumer

import (
	"context"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/sujal-lgtm/Contextify/backend/services/anomaly/internal/db"
	"github.com/sujal-lgtm/Contextify/backend/services/anomaly/internal/detector"
)

// Pass DB connection to consumer
var dbConn *db.DB

func Init(db *db.DB) {
	dbConn = db
}

func Start() error {
	if dbConn == nil {
		logrus.Fatal("DB connection not initialized. Call Init(db) first.")
	}

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

	// Error rate tracker (window: 10 seconds)
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

		// 1️⃣ Persist context to DB
		if err := dbConn.SaveContext(db.Event{
			TraceID:     event.TraceID,
			Service:     event.Service,
			Timestamp:   event.Timestamp,
			LatencyMs:   event.LatencyMs,
			Status:      event.Status,
			QueueLength: event.QueueLength,
		}); err != nil {
			logrus.Errorf("Failed to save context: %v", err)
		}

		// 2️⃣ Add event to error rate tracker
		tracker.AddEvent(*event)

		anomaliesDetected := false

		// 3️⃣ Check latency spike
		if detector.CheckLatency(*event, 300) {
			detector.PersistAnomaly(dbConn, *event, "latency_spike", 0)
			msg := detector.CreateAnomalyMessage(*event, "latency_spike")
			writeKafkaMessage(w, msg)
			anomaliesDetected = true
		}

		// 4️⃣ Check error rate spike
		if tracker.CheckErrorRate(0.5) {
			detector.PersistAnomaly(dbConn, *event, "error_rate_spike", 0.5)
			msg := detector.CreateAnomalyMessage(*event, "error_rate_spike")
			writeKafkaMessage(w, msg)
			anomaliesDetected = true
		}

		// 5️⃣ Check queue length threshold
		if detector.CheckQueueLength(*event, 100) {
			detector.PersistAnomaly(dbConn, *event, "queue_length_spike", 0)
			msg := detector.CreateAnomalyMessage(*event, "queue_length_spike")
			writeKafkaMessage(w, msg)
			anomaliesDetected = true
		}

		if anomaliesDetected {
			logrus.Warnf("🚨 Anomaly detected for service: %s", event.Service)
			detector.IncrementAnomalyCount()
		}
	}
}

// Helper to write anomaly to Kafka
func writeKafkaMessage(w *kafka.Writer, msg string) {
	err := w.WriteMessages(context.Background(), kafka.Message{
		Value: []byte(msg),
	})
	if err != nil {
		logrus.Errorf("Failed to write anomaly to Kafka: %v", err)
	} else {
		logrus.Infof("⚠️ Anomaly published to Kafka: %s", msg)
	}
}
