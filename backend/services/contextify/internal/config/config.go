package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	RestPort string
	GrpcPort string
}

func LoadConfig() (*Config, error) {
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

	return &Config{
		RestPort: restPort,
		GrpcPort: grpcPort,
	}, nil
}
