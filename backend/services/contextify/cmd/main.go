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
	cfg := config.LoadConfig()

	router := mux.NewRouter()
	router.HandleFunc("/health", handlers.HealthHandler).Methods("GET")

	httpServer := &http.Server{
		Addr:    ":" + cfg.RestPort,
		Handler: router,
	}

	stopGrpc := grpcserver.StartGrpcServer(cfg.GrpcPort)

	go func() {
		logrus.Infof("REST server listening on :%s", cfg.RestPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("REST server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logrus.Info("Shutting down servers...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		logrus.Errorf("REST shutdown error: %v", err)
	}

	stopGrpc()

	logrus.Info("Shutdown complete.")
}
