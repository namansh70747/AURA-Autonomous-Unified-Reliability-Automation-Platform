-- AURA Database Schema
-- This runs automatically when PostgreSQL starts

-- Metrics table (stores raw metrics from Prometheus)
CREATE TABLE IF NOT EXISTS metrics (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    service_name VARCHAR(100) NOT NULL,
    metric_name VARCHAR(100) NOT NULL,
    metric_value FLOAT NOT NULL,
    labels JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Events table (stores Kubernetes events)
CREATE TABLE IF NOT EXISTS events (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    event_type VARCHAR(50) NOT NULL,
    pod_name VARCHAR(200),
    namespace VARCHAR(100),
    message TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Decisions table (stores AURA decisions)
CREATE TABLE IF NOT EXISTS decisions (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    pattern_detected VARCHAR(100),
    action_type VARCHAR(50) NOT NULL,
    confidence FLOAT NOT NULL,
    reason TEXT,
    parameters JSONB,
    executed BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX idx_metrics_timestamp ON metrics(timestamp DESC);
CREATE INDEX idx_metrics_service ON metrics(service_name);
CREATE INDEX idx_events_timestamp ON events(timestamp DESC);
CREATE INDEX idx_decisions_timestamp ON decisions(timestamp DESC);

-- Insert test data
INSERT INTO metrics (service_name, metric_name, metric_value, labels) VALUES
('sample-app', 'cpu_usage', 45.0, '{"pod": "sample-app-1"}'),
('sample-app', 'memory_usage', 60.0, '{"pod": "sample-app-1"}');

COMMENT ON TABLE metrics IS 'Raw metrics from Prometheus';
COMMENT ON TABLE events IS 'Kubernetes events from API';
COMMENT ON TABLE decisions IS 'AURA autonomous decisions';
