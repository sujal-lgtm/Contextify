package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/sujal-lgtm/Contextify/backend/services/contextify/internal/config"
	"github.com/sujal-lgtm/Contextify/backend/services/contextify/internal/grpcserver"
	"github.com/sujal-lgtm/Contextify/backend/services/contextify/internal/handlers"
)

func main() {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.Info("🚀 Contextify service starting...")

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		logrus.Fatalf("Failed to load config: %v", err)
	}

	// Create router
	router := mux.NewRouter()

	// Add middleware
	router.Use(loggingMiddleware)
	router.Use(corsMiddleware)

	// Health check endpoint
	router.HandleFunc("/health", handlers.HealthHandler).Methods("GET")

	// Incident management endpoints
	router.HandleFunc("/incident/start", handlers.StartIncidentHandler).Methods("POST")
	router.HandleFunc("/incident/stop", handlers.StopIncidentHandler).Methods("POST")

	// Metrics endpoint
	router.HandleFunc("/metrics", handlers.MetricsHandler).Methods("GET")

	// Start gRPC server
	grpcShutdown := grpcserver.StartGrpcServer(cfg.GrpcPort)
	defer grpcShutdown()

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + cfg.RestPort,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start HTTP server in goroutine
	go func() {
		logrus.Infof("🌍 HTTP server listening on :%s", cfg.RestPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logrus.Info("🛑 Shutting down Contextify service...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logrus.Errorf("HTTP server forced to shutdown: %v", err)
	}

	logrus.Info("✅ Contextify service stopped gracefully")
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
