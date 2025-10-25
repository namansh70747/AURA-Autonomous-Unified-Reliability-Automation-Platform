package storage

import (
	"encoding/json"
	"time"
)

// Metric represents a time-series metric data point
type Metric struct {
	ID          int64           `json:"id"`
	Timestamp   time.Time       `json:"timestamp"`
	ServiceName string          `json:"service_name"`
	MetricName  string          `json:"metric_name"`
	MetricValue float64         `json:"metric_value"`
	Labels      json.RawMessage `json:"labels,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}

// MetricStats represents statistical analysis of metrics
type MetricStats struct {
	ServiceName string        `json:"service_name"`
	MetricName  string        `json:"metric_name"`
	Count       int64         `json:"count"`
	Avg         float64       `json:"avg"`
	Min         float64       `json:"min"`
	Max         float64       `json:"max"`
	StdDev      float64       `json:"stddev"`
	Duration    time.Duration `json:"duration"`
}

// Event represents a Kubernetes event
type Event struct {
	ID        int64     `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	EventType string    `json:"event_type"`
	PodName   string    `json:"pod_name"`
	Namespace string    `json:"namespace"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

// Decision represents an AURA decision
type Decision struct {
	ID              int64           `json:"id"`
	Timestamp       time.Time       `json:"timestamp"`
	PatternDetected string          `json:"pattern_detected"`
	ActionType      string          `json:"action_type"`
	Confidence      float64         `json:"confidence"`
	Reason          string          `json:"reason"`
	Parameters      json.RawMessage `json:"parameters,omitempty"`
	Executed        bool            `json:"executed"`
	CreatedAt       time.Time       `json:"created_at"`
}

// DecisionStats represents decision statistics
type DecisionStats struct {
	Total         int64   `json:"total"`
	Executed      int64   `json:"executed"`
	Pending       int64   `json:"pending"`
	AvgConfidence float64 `json:"avg_confidence"`
}
