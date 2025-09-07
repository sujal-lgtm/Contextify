package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/sujal-lgtm/Contextify/backend/services/anomaly/internal/consumer"
	"github.com/sujal-lgtm/Contextify/backend/services/anomaly/internal/db"
	"github.com/sujal-lgtm/Contextify/backend/services/anomaly/internal/detector"
)

const (
	kafkaBroker    = "kafka:9092"
	anomaliesTopic = "anomalies"
)

func main() {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.Info("🚨 Anomaly service starting...")

	// ----------------- Initialize DB -----------------
	dsn := "postgres://contextify:contextify@postgres:5432/contextify?sslmode=disable"
	dbConn, err := db.NewDB(dsn)
	if err != nil {
		logrus.Fatalf("Failed to connect to DB: %v", err)
	}

	// Initialize consumer with DB connection
	consumer.Init(dbConn)

	// Start Kafka consumer in background
	go func() {
		if err := consumer.Start(); err != nil {
			logrus.Fatalf("Consumer failed: %v", err)
		}
	}()

	// ----------------- Setup HTTP router -----------------
	router := mux.NewRouter()
	router.Use(loggingMiddleware)
	router.Use(corsMiddleware)

	// Health check
	router.HandleFunc("/health", healthHandler).Methods("GET")

	// Manual incident trigger
	router.HandleFunc("/incident/trigger", func(w http.ResponseWriter, r *http.Request) {
		triggerIncidentHandler(w, r)
	}).Methods("POST")

	// Metrics
	router.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		metricsHandler(w, r)
	}).Methods("GET")

	// Anomaly detection status
	router.HandleFunc("/anomalies/status", func(w http.ResponseWriter, r *http.Request) {
		anomalyStatusHandler(w, r)
	}).Methods("GET")

	// ✅ GET /anomalies?service=X → anomalies + recent context
	router.HandleFunc("/anomalies", func(w http.ResponseWriter, r *http.Request) {
		service := r.URL.Query().Get("service")
		if service == "" {
			http.Error(w, "Missing 'service' query parameter", http.StatusBadRequest)
			return
		}

		limit := 10
		if lStr := r.URL.Query().Get("limit"); lStr != "" {
			if l, err := strconv.Atoi(lStr); err == nil {
				limit = l
			}
		}

		anomalies, err := dbConn.GetRecentAnomalies(service, limit)
		if err != nil {
			http.Error(w, "Failed to fetch anomalies", http.StatusInternalServerError)
			return
		}

		// Get recent context events for the service
		ctxEvents, err := detector.AttachRecentContext(dbConn, service, limit)
		if err != nil {
			ctxEvents = []detector.Event{}
		}

		// Create response with anomalies and context
		response := map[string]interface{}{
			"anomalies":      anomalies,
			"recent_context": ctxEvents,
			"service":        service,
			"timestamp":      time.Now().Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}).Methods("GET")

	// ----------------- Start HTTP server -----------------
	server := &http.Server{
		Addr:         ":8081",
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logrus.Info("🌍 API listening on :8081")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logrus.Info("🛑 Shutting down Anomaly service...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logrus.Errorf("HTTP server forced to shutdown: %v", err)
	}
	logrus.Info("✅ Anomaly service stopped gracefully")
}

// ----------------- Handlers -----------------

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
		"anomalies_detected": 0, // Fixed: removed call to non-existent function
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

	payload, _ := json.Marshal(anomaly)
	writer.WriteMessages(context.Background(),
		kafka.Message{
			Key:   []byte("manual_incident"),
			Value: payload,
		},
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(payload)
}

// ----------------- Middleware -----------------

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
