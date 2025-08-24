package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"github.com/sujal-lgtm/Contextify/backend/services/anomaly/internal/consumer"
	"github.com/sujal-lgtm/Contextify/backend/services/anomaly/internal/detector"
)

const (
	kafkaBroker    = "kafka:9092"
	eventsTopic    = "contextify-events"
	anomaliesTopic = "anomalies"
)

func main() {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.Info("🚨 Anomaly service starting...")

	// Start consumer in background
	go func() {
		if err := consumer.Start(); err != nil {
			logrus.Fatalf("Consumer failed: %v", err)
		}
	}()

	// Create router
	router := mux.NewRouter()

	// Add middleware
	router.Use(loggingMiddleware)
	router.Use(corsMiddleware)

	// Health check
	router.HandleFunc("/health", healthHandler).Methods("GET")

	// Incident trigger endpoint
	router.HandleFunc("/incident/trigger", triggerIncidentHandler).Methods("POST")

	// Metrics endpoint
	router.HandleFunc("/metrics", metricsHandler).Methods("GET")

	// Anomaly detection status
	router.HandleFunc("/anomalies/status", anomalyStatusHandler).Methods("GET")

	// Create HTTP server
	server := &http.Server{
		Addr:         ":8081",
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start HTTP server in goroutine
	go func() {
		logrus.Info("🌍 API listening on :8081")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logrus.Info("🛑 Shutting down Anomaly service...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logrus.Errorf("HTTP server forced to shutdown: %v", err)
	}

	logrus.Info("✅ Anomaly service stopped gracefully")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"service": "anomaly-detector",
	})
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	metrics := map[string]interface{}{
		"service":            "anomaly-detector",
		"status":             "healthy",
		"anomalies_detected": detector.GetAnomalyCount(),
		"timestamp":          time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

func anomalyStatusHandler(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"service":          "anomaly-detector",
		"detection_active": true,
		"thresholds": map[string]interface{}{
			"latency_ms":   300,
			"error_rate":   0.5,
			"queue_length": 100,
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// handle manual incident trigger
func triggerIncidentHandler(w http.ResponseWriter, r *http.Request) {
	writer := &kafka.Writer{
		Addr:     kafka.TCP(kafkaBroker),
		Topic:    anomaliesTopic,
		Balancer: &kafka.LeastBytes{},
	}
	defer writer.Close()

	anomaly := map[string]interface{}{
		"type":        "manual_incident",
		"service":     "contextify",
		"status":      "anomaly_detected",
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"severity":    "high",
		"description": "Manually triggered incident for testing",
	}

	payload, err := json.Marshal(anomaly)
	if err != nil {
		logrus.Errorf("Failed to marshal anomaly: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	err = writer.WriteMessages(context.Background(),
		kafka.Message{
			Key:   []byte("manual_incident"),
			Value: payload,
		},
	)
	if err != nil {
		logrus.Errorf("Failed to write anomaly: %v", err)
		http.Error(w, "Failed to write anomaly", http.StatusInternalServerError)
		return
	}

	logrus.Info("🚨 Published manual anomaly:", string(payload))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(payload)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logrus.WithFields(logrus.Fields{
			"method":     r.Method,
			"path":       r.URL.Path,
			"duration":   time.Since(start),
			"user_agent": r.UserAgent(),
		}).Info("HTTP request")
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
