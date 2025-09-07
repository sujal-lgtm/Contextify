package detector

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/sujal-lgtm/Contextify/backend/services/anomaly/internal/db"
)

type Event struct {
	TraceID     string `json:"trace_id"`
	Service     string `json:"service"`
	Timestamp   int64  `json:"timestamp"`
	LatencyMs   int    `json:"latency_ms"`
	Status      string `json:"status"`
	QueueLength int    `json:"queue_length"`
}

// ErrorRateTracker tracks error rates over a time window
type ErrorRateTracker struct {
	window     time.Duration
	events     []Event
	errorCount int
}

// NewErrorRateTracker creates a new error rate tracker
func NewErrorRateTracker(windowSeconds int) *ErrorRateTracker {
	return &ErrorRateTracker{
		window: time.Duration(windowSeconds) * time.Second,
		events: make([]Event, 0),
	}
}

// AddEvent adds an event to the tracker
func (t *ErrorRateTracker) AddEvent(event Event) {
	now := time.Now()

	// Remove old events outside the window
	t.events = append(t.events, event)

	// Keep only events within the window
	var recentEvents []Event
	for _, e := range t.events {
		if now.Sub(time.UnixMilli(e.Timestamp)) <= t.window {
			recentEvents = append(recentEvents, e)
		}
	}
	t.events = recentEvents

	// Count errors in recent events
	t.errorCount = 0
	for _, e := range t.events {
		if e.Status != "success" {
			t.errorCount++
		}
	}
}

// CheckErrorRate checks if error rate exceeds threshold
func (t *ErrorRateTracker) CheckErrorRate(threshold float64) bool {
	if len(t.events) == 0 {
		return false
	}
	errorRate := float64(t.errorCount) / float64(len(t.events))
	return errorRate > threshold
}

// CheckLatency checks if latency exceeds threshold
func CheckLatency(event Event, thresholdMs int) bool {
	return event.LatencyMs > thresholdMs
}

// CheckQueueLength checks if queue length exceeds threshold
func CheckQueueLength(event Event, threshold int) bool {
	return event.QueueLength > threshold
}

// CreateAnomalyMessage creates a JSON message for anomaly
func CreateAnomalyMessage(event Event, anomalyType string) string {
	anomaly := map[string]interface{}{
		"type":         anomalyType,
		"trace_id":     event.TraceID,
		"service":      event.Service,
		"timestamp":    event.Timestamp,
		"latency_ms":   event.LatencyMs,
		"status":       event.Status,
		"queue_length": event.QueueLength,
	}

	msg, err := json.Marshal(anomaly)
	if err != nil {
		logrus.Errorf("Failed to marshal anomaly message: %v", err)
		return ""
	}
	return string(msg)
}

// IncrementAnomalyCount increments the global anomaly counter
func IncrementAnomalyCount() {
	// This could be implemented with a global counter or metrics
	// For now, just log it
	logrus.Info("Anomaly count incremented")
}

// ParseEvent parses JSON event from Kafka message
func ParseEvent(data []byte) (*Event, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	e := &Event{}

	if v, ok := raw["trace_id"].(string); ok {
		e.TraceID = v
	}
	if v, ok := raw["service"].(string); ok {
		e.Service = v
	}
	if v, ok := raw["latency_ms"].(float64); ok {
		e.LatencyMs = int(v)
	}
	if v, ok := raw["status"].(string); ok {
		e.Status = v
	}
	if v, ok := raw["queue_length"].(float64); ok {
		e.QueueLength = int(v)
	}

	// Handle timestamp (could be number, string millis, or RFC3339 string)
	switch ts := raw["timestamp"].(type) {
	case float64:
		e.Timestamp = int64(ts)
	case string:
		// Try parse as integer millis
		if millis, err := strconv.ParseInt(ts, 10, 64); err == nil {
			e.Timestamp = millis
		} else {
			// Try parse as RFC3339
			if parsed, err := time.Parse(time.RFC3339, ts); err == nil {
				e.Timestamp = parsed.UnixMilli()
			} else {
				return nil, err
			}
		}
	}

	return e, nil
}

// Persist anomaly to DB
func PersistAnomaly(dbConn *db.DB, event Event, anomalyType string, errorRate float64) {
	a := db.Anomaly{
		Type:        anomalyType,
		Service:     event.Service,
		LatencyMs:   event.LatencyMs,
		ErrorRate:   errorRate,
		QueueLength: event.QueueLength,
		Timestamp:   time.Now().UnixMilli(),
	}

	if err := dbConn.SaveAnomaly(a); err != nil {
		logrus.Errorf("Failed to persist anomaly: %v", err)
	}
}

// Fetch last N events for a service
func AttachRecentContext(dbConn *db.DB, service string, limit int) ([]Event, error) {
	dbEvents, err := dbConn.GetRecentEvents(service, limit)
	if err != nil {
		logrus.Errorf("Failed to fetch recent context: %v", err)
		return nil, err
	}

	events := make([]Event, len(dbEvents))
	for i, dbEvent := range dbEvents {
		events[i] = Event{
			Service:     dbEvent.Service,
			Timestamp:   dbEvent.Timestamp,
			LatencyMs:   dbEvent.LatencyMs,
			Status:      dbEvent.Status,
			QueueLength: dbEvent.QueueLength,
		}
	}
	return events, nil
}
