package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// StartIncidentHandler handles the POST /incident/start endpoint
func StartIncidentHandler(w http.ResponseWriter, r *http.Request) {
	logrus.Info("🚨 Manual incident triggered")

	// Create incident response
	response := map[string]interface{}{
		"status":      "incident_triggered",
		"service":     "contextify",
		"timestamp":   time.Now().Format(time.RFC3339),
		"message":     "Manual incident triggered successfully",
		"incident_id": generateIncidentID(),
		"severity":    "medium",
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// StopIncidentHandler handles the POST /incident/stop endpoint
func StopIncidentHandler(w http.ResponseWriter, r *http.Request) {
	logrus.Info("✅ Manual incident stopped")

	response := map[string]interface{}{
		"status":    "incident_resolved",
		"service":   "contextify",
		"timestamp": time.Now().Format(time.RFC3339),
		"message":   "Manual incident resolved successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// MetricsHandler handles the GET /metrics endpoint
func MetricsHandler(w http.ResponseWriter, r *http.Request) {
	metrics := map[string]interface{}{
		"service":          "contextify",
		"status":           "healthy",
		"uptime":           getUptime(),
		"version":          "1.0.0",
		"timestamp":        time.Now().Format(time.RFC3339),
		"active_incidents": 0,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

func generateIncidentID() string {
	return fmt.Sprintf("inc_%d", time.Now().Unix())
}

func getUptime() string {
	// This would typically come from a service that tracks startup time
	return "1h 23m 45s"
}
