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

-- Diagnoses table (stores pattern analysis results)
CREATE TABLE IF NOT EXISTS diagnoses (
    id SERIAL PRIMARY KEY,
    service_name VARCHAR(100) NOT NULL,
    problem_type VARCHAR(100) NOT NULL,
    confidence DECIMAL(5,2) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    evidence JSONB,
    recommendation TEXT,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- AI-Level Analyzer Tables (Phase 2.5 - Ultimate Diagnosis)

-- Ultimate diagnoses with comprehensive AI features
CREATE TABLE IF NOT EXISTS ultimate_diagnoses (
    id BIGSERIAL PRIMARY KEY,
    service_name VARCHAR(255) NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    analysis_duration BIGINT,
    
    -- Features (stored as JSONB for flexibility)
    features JSONB,
    
    -- Primary Detection
    primary_problem VARCHAR(100),
    primary_detected BOOLEAN,
    primary_confidence DOUBLE PRECISION,
    primary_severity VARCHAR(50),
    primary_evidence JSONB,
    
    -- All Detections
    all_detections JSONB,
    
    -- Scores
    health_score DOUBLE PRECISION,
    stability_index DOUBLE PRECISION,
    predictability_score DOUBLE PRECISION,
    system_stress DOUBLE PRECISION,
    
    -- Decision Support
    risk_level VARCHAR(50),
    action_required BOOLEAN,
    predictive_insights JSONB,
    recommendation TEXT,
    
    -- Traceability
    prediction_id VARCHAR(255) UNIQUE
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_metrics_timestamp ON metrics(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_metrics_service ON metrics(service_name);
CREATE INDEX IF NOT EXISTS idx_metrics_composite ON metrics(service_name, metric_name, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_decisions_timestamp ON decisions(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_diagnoses_service ON diagnoses(service_name);
CREATE INDEX IF NOT EXISTS idx_diagnoses_timestamp ON diagnoses(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_diagnoses_severity ON diagnoses(severity);

-- AI-Level indexes
CREATE INDEX IF NOT EXISTS idx_ultimate_diagnoses_service ON ultimate_diagnoses(service_name);
CREATE INDEX IF NOT EXISTS idx_ultimate_diagnoses_timestamp ON ultimate_diagnoses(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_ultimate_diagnoses_risk ON ultimate_diagnoses(risk_level);
CREATE INDEX IF NOT EXISTS idx_ultimate_diagnoses_action ON ultimate_diagnoses(action_required);
CREATE INDEX IF NOT EXISTS idx_ultimate_diagnoses_prediction ON ultimate_diagnoses(prediction_id);

-- Create views for analytics
CREATE OR REPLACE VIEW service_health_trends AS
SELECT 
    service_name,
    DATE_TRUNC('hour', timestamp) as hour,
    AVG(health_score) as avg_health_score,
    AVG(stability_index) as avg_stability,
    AVG(system_stress) as avg_stress,
    COUNT(*) as diagnosis_count,
    SUM(CASE WHEN action_required = true THEN 1 ELSE 0 END) as actions_required
FROM ultimate_diagnoses
WHERE timestamp > NOW() - INTERVAL '24 hours'
GROUP BY service_name, DATE_TRUNC('hour', timestamp)
ORDER BY hour DESC;

CREATE OR REPLACE VIEW recent_critical_issues AS
SELECT 
    service_name,
    timestamp,
    primary_problem,
    primary_confidence,
    risk_level,
    recommendation
FROM ultimate_diagnoses
WHERE risk_level IN ('CRITICAL', 'HIGH')
  AND timestamp > NOW() - INTERVAL '6 hours'
ORDER BY timestamp DESC;

-- Insert test data
INSERT INTO metrics (service_name, metric_name, metric_value, labels) VALUES
('sample-app', 'cpu_usage', 45.0, '{"pod": "sample-app-1"}'),
('sample-app', 'memory_usage', 60.0, '{"pod": "sample-app-1"}'),
('sample-app', 'cpu_usage_percent', 45.0, '{"pod": "sample-app-1"}'),
('sample-app', 'memory_usage_percent', 60.0, '{"pod": "sample-app-1"}')
ON CONFLICT DO NOTHING;

COMMENT ON TABLE metrics IS 'Raw metrics from Prometheus';
COMMENT ON TABLE events IS 'Kubernetes events from API';
COMMENT ON TABLE decisions IS 'AURA autonomous decisions';
COMMENT ON TABLE diagnoses IS 'Pattern analysis results (Phase 2)';
COMMENT ON TABLE ultimate_diagnoses IS 'AI-level comprehensive diagnostic results (Phase 2.5)';
COMMENT ON VIEW service_health_trends IS 'Health trends over time for all services';
COMMENT ON VIEW recent_critical_issues IS 'Recent critical/high severity issues requiring attention';
