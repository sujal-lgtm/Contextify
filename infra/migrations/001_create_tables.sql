-- Contextify + Anomaly Detection Schema

-- Stores raw context events (latency, status, queue length etc.)
CREATE TABLE contexts (
    id SERIAL PRIMARY KEY,
    service TEXT NOT NULL,
    timestamp TIMESTAMP NOT NULL DEFAULT NOW(),
    latency_ms INT,
    status TEXT,
    queue_length INT
);

-- Stores anomalies detected from rules (latency, error %, queue length)
CREATE TABLE anomalies (
    id SERIAL PRIMARY KEY,
    type TEXT NOT NULL,
    service TEXT NOT NULL,
    latency_ms INT,
    error_rate DOUBLE PRECISION,
    queue_length INT,
    timestamp TIMESTAMP NOT NULL DEFAULT NOW()
);
