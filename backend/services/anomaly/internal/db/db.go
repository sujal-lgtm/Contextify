package db

import (
	"database/sql"

	_ "github.com/lib/pq"
)

type DB struct {
	Conn *sql.DB
}

// Initialize DB connection
func NewDB(dsn string) (*DB, error) {
	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	if err := conn.Ping(); err != nil {
		return nil, err
	}
	return &DB{Conn: conn}, nil
}

// Save a context event
func (db *DB) SaveContext(event Event) error {
	_, err := db.Conn.Exec(
		`INSERT INTO contexts(trace_id, service, timestamp, latency_ms, status, queue_length)
         VALUES($1, $2, to_timestamp($3/1000.0), $4, $5, $6)`,
		event.TraceID, event.Service, event.Timestamp, event.LatencyMs, event.Status, event.QueueLength,
	)
	return err
}

// Save an anomaly
func (db *DB) SaveAnomaly(a Anomaly) error {
	_, err := db.Conn.Exec(
		`INSERT INTO anomalies(type, service, latency_ms, error_rate, queue_length, timestamp)
		 VALUES($1, $2, $3, $4, $5, to_timestamp($6/1000.0))`,
		a.Type, a.Service, a.LatencyMs, a.ErrorRate, a.QueueLength, a.Timestamp,
	)
	return err
}

// Fetch recent anomalies
func (db *DB) GetRecentAnomalies(service string, limit int) ([]Anomaly, error) {
	rows, err := db.Conn.Query(
		`SELECT type, service, latency_ms, error_rate, queue_length, extract(epoch from timestamp)*1000 as ts
		 FROM anomalies
		 WHERE service = $1
		 ORDER BY timestamp DESC
		 LIMIT $2`, service, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var anomalies []Anomaly
	for rows.Next() {
		var a Anomaly
		var ts float64
		if err := rows.Scan(&a.Type, &a.Service, &a.LatencyMs, &a.ErrorRate, &a.QueueLength, &ts); err != nil {
			return nil, err
		}
		a.Timestamp = int64(ts)
		anomalies = append(anomalies, a)
	}
	return anomalies, nil
}

// Fetch recent context events
func (db *DB) GetRecentEvents(service string, limit int) ([]Event, error) {
	rows, err := db.Conn.Query(
		`SELECT service, extract(epoch from timestamp)*1000 as ts, latency_ms, status, queue_length
		 FROM contexts
		 WHERE service = $1
		 ORDER BY timestamp DESC
		 LIMIT $2`, service, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event
		var ts float64
		if err := rows.Scan(&e.Service, &ts, &e.LatencyMs, &e.Status, &e.QueueLength); err != nil {
			return nil, err
		}
		e.Timestamp = int64(ts)
		events = append(events, e)
	}
	return events, nil
}

// DB structs
type Event struct {
	TraceID     string
	Service     string
	Timestamp   int64
	LatencyMs   int
	Status      string
	QueueLength int
}

type Anomaly struct {
	Type        string
	Service     string
	LatencyMs   int
	ErrorRate   float64
	QueueLength int
	Timestamp   int64
}
