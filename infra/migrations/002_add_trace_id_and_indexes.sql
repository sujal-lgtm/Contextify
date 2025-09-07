-- Add trace_id to contexts
ALTER TABLE contexts
ADD COLUMN trace_id TEXT NOT NULL DEFAULT '';

-- Index for fast lookups by trace_id + timestamp (newest first)
CREATE INDEX IF NOT EXISTS idx_contexts_trace_time
ON contexts(trace_id, timestamp DESC);

-- Index for anomalies: service + timestamp (for filtering recent anomalies by service)
CREATE INDEX IF NOT EXISTS idx_anomalies_service_time
ON anomalies(service, timestamp DESC);
