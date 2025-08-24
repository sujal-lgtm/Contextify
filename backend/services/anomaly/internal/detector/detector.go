package detector

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Event represents a single Contextify event
type Event struct {
	Service     string `json:"service"`
	Timestamp   int64  `json:"timestamp"`
	LatencyMs   int    `json:"latency_ms"`
	Status      string `json:"status"`       // "success" or "error"
	QueueLength int    `json:"queue_length"` // current queue length
}

// AnomalyThresholds defines thresholds for anomaly detection
type AnomalyThresholds struct {
	LatencyMs   int
	ErrorRate   float64
	QueueLength int
}

// Global anomaly counter
var (
	anomalyCount int64
	anomalyMutex sync.RWMutex
)

// ParseEvent parses raw JSON into Event struct
func ParseEvent(raw []byte) (Event, error) {
	var e Event
	err := json.Unmarshal(raw, &e)
	if err != nil {
		logrus.Errorf("Failed to parse event: %v", err)
		return Event{}, err
	}
	return e, nil
}

// CreateAnomalyMessage generates a simple string for Kafka
func CreateAnomalyMessage(event Event, anomalyType string) string {
	anomaly := map[string]interface{}{
		"type":         anomalyType,
		"service":      event.Service,
		"latency_ms":   event.LatencyMs,
		"queue_length": event.QueueLength,
		"status":       event.Status,
		"timestamp":    time.Now().Unix(),
		"severity":     determineSeverity(anomalyType, event),
	}

	payload, _ := json.Marshal(anomaly)
	return string(payload)
}

func determineSeverity(anomalyType string, event Event) string {
	switch anomalyType {
	case "latency_spike":
		if event.LatencyMs > 1000 {
			return "critical"
		}
		return "warning"
	case "error_rate_spike":
		return "critical"
	case "queue_length_spike":
		if event.QueueLength > 500 {
			return "critical"
		}
		return "warning"
	default:
		return "medium"
	}
}

// ----------------- Error Rate Spike -----------------

type ErrorRateTracker struct {
	mu        sync.RWMutex
	events    []Event
	windowSec int
}

func NewErrorRateTracker(windowSec int) *ErrorRateTracker {
	return &ErrorRateTracker{
		events:    []Event{},
		windowSec: windowSec,
	}
}

// AddEvent adds an event to the tracker and removes old events
func (t *ErrorRateTracker) AddEvent(e Event) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now().Unix()
	t.events = append(t.events, e)

	// remove events older than window
	var filtered []Event
	for _, ev := range t.events {
		// Fix: ev.Timestamp is in milliseconds, convert to seconds
		if (now - ev.Timestamp/1000) <= int64(t.windowSec) {
			filtered = append(filtered, ev)
		}
	}
	t.events = filtered
}

// CheckErrorRate returns true if error rate exceeds threshold
func (t *ErrorRateTracker) CheckErrorRate(threshold float64) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if len(t.events) == 0 {
		return false
	}

	errorCount := 0
	for _, ev := range t.events {
		if ev.Status == "error" {
			errorCount++
		}
	}

	rate := float64(errorCount) / float64(len(t.events))
	return rate >= threshold
}

// ----------------- Queue Length Threshold -----------------

func CheckQueueLength(e Event, threshold int) bool {
	return e.QueueLength > threshold
}

// ----------------- Latency -----------------

func CheckLatency(e Event, threshold int) bool {
	return e.LatencyMs > threshold
}

// GetAnomalyCount returns the current anomaly count
func GetAnomalyCount() int64 {
	anomalyMutex.RLock()
	defer anomalyMutex.RUnlock()
	return anomalyCount
}

// IncrementAnomalyCount increments the anomaly counter
func IncrementAnomalyCount() {
	anomalyMutex.Lock()
	defer anomalyMutex.Unlock()
	anomalyCount++
}

// ResetAnomalyCount resets the anomaly counter (useful for testing)
func ResetAnomalyCount() {
	anomalyMutex.Lock()
	defer anomalyMutex.Unlock()
	anomalyCount = 0
}
