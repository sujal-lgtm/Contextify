package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	RestPort     string
	GrpcPort     string
	KafkaBrokers string
	DatabaseURL  string
}

func LoadConfig() (*Config, error) {
	// Load .env if available
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: could not load .env file: %v", err)
	}

	restPort := os.Getenv("REST_PORT")
	if restPort == "" {
		restPort = "8080"
	}

	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "50051"
	}

	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	if kafkaBrokers == "" {
		// default: inside docker-compose, "kafka" is the service name
		kafkaBrokers = "kafka:9092"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		// default Postgres connection inside Docker
		databaseURL = "postgres://contextify:contextify@postgres:5432/contextify?sslmode=disable"
	}

	return &Config{
		RestPort:     restPort,
		GrpcPort:     grpcPort,
		KafkaBrokers: kafkaBrokers,
		DatabaseURL:  databaseURL,
	}, nil
}
