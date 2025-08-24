# Contextify - Event-Driven Anomaly Detection System

A robust, production-ready event-driven system for detecting anomalies in microservices with real-time monitoring and alerting capabilities.

## ��️ Architecture

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│  Contextify │───▶│    Kafka    │───▶│   Anomaly  │
│   Service   │    │   Broker    │    │  Detector   │
└─────────────┘    └─────────────┘    └─────────────┘
       │                   │                   │
       │                   │                   │
       ▼                   ▼                   ▼
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   gRPC API  │    │  Zookeeper  │    │   Producer  │
│   (50051)   │    │             │    │  (Testing)  │
└─────────────┘    └─────────────┘    └─────────────┘
```

## Features

- **Real-time Event Processing**: Kafka-based event streaming
- **Anomaly Detection**: Multi-threshold anomaly detection
- **gRPC & HTTP APIs**: Dual protocol support
- **Health Monitoring**: Comprehensive health checks
- **Graceful Shutdown**: Proper service lifecycle management
- **Production Ready**: Security, logging, and monitoring
- **Docker Support**: Complete containerization

## 🛠️ Services

###
