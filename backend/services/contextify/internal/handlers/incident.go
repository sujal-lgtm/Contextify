package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/sujal-lgtm/Contextify/backend/pkg/producer"

	"github.com/sirupsen/logrus"
)

var (
	publisher = producer.NewPublisher([]string{"kafka:9092"}, "contextify-events")
)

type IncidentRequest struct {
	Service string `json:"service"`
	Error   string `json:"error"`
	Latency int64  `json:"latency_ms"`
}

func StartIncidentHandler(w http.ResponseWriter, r *http.Request) {
	var req IncidentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	// Build event for Kafka
	event := map[string]interface{}{
		"service":  req.Service,
		"error":    req.Error,
		"latency":  req.Latency,
		"detected": time.Now().UTC(),
	}

	// Publish event to Kafka
	if err := publisher.PublishEvent(context.Background(), req.Service, event); err != nil {
		logrus.Errorf("failed to publish incident event: %v", err)
		http.Error(w, "failed to publish event", http.StatusInternalServerError)
		return
	}

	// Respond as API
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "incident started",
		"service": req.Service,
	})
}

func StopIncidentHandler(w http.ResponseWriter, r *http.Request) {
	var req IncidentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	// Build event for Kafka
	event := map[string]interface{}{
		"service":  req.Service,
		"error":    req.Error,
		"latency":  req.Latency,
		"resolved": time.Now().UTC(),
		"action":   "stop",
	}

	// Publish event to Kafka
	if err := publisher.PublishEvent(context.Background(), req.Service, event); err != nil {
		logrus.Errorf("failed to publish incident stop event: %v", err)
		http.Error(w, "failed to publish event", http.StatusInternalServerError)
		return
	}

	// Respond as API
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "incident stopped",
		"service": req.Service,
	})
}

func MetricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "ok",
		"service":   "contextify",
		"timestamp": time.Now().UTC(),
		"endpoints": map[string]string{
			"health":         "/health",
			"incident_start": "/incident/start",
			"incident_stop":  "/incident/stop",
			"metrics":        "/metrics",
		},
	})
}
