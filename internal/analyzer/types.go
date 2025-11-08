package analyzer

import (
	"time"
)

type DetectionType string

const (
	DetectionMemoryLeak         DetectionType = "MEMORY_LEAK"
	DetectionDeploymentBug      DetectionType = "DEPLOYMENT_BUG"
	DetectionCascadingFailure   DetectionType = "CASCADING_FAILURE"
	DetectionExternalFailure    DetectionType = "EXTERNAL_FAILURE"
	DetectionResourceExhaustion DetectionType = "RESOURCE_EXHAUSTION"
	DetectionHealthy            DetectionType = "HEALTHY"
	DetectionUnknown            DetectionType = "UNKNOWN"
)

type Detection struct {
	Type           DetectionType          `json:"type"`
	ServiceName    string                 `json:"service_name"`
	Detected       bool                   `json:"detected"`
	Confidence     float64                `json:"confidence"`
	Timestamp      time.Time              `json:"timestamp"`
	Evidence       map[string]interface{} `json:"evidence"`
	Recommendation string                 `json:"recommendation"`
	Severity       string                 `json:"severity"` // LOW, MEDIUM, HIGH, CRITICAL
}

type Diagnosis struct {
	ServiceName         string                 `json:"service_name"`
	Problem             DetectionType          `json:"problem"`
	Confidence          float64                `json:"confidence"`
	Timestamp           time.Time              `json:"timestamp"`
	Evidence            map[string]interface{} `json:"evidence"`
	Recommendation      string                 `json:"recommendation"`
	Severity            string                 `json:"severity"`
	AllDetections       []Detection            `json:"all_detections,omitempty"`
	MultipleProblems    bool                   `json:"multiple_problems"`
	HighConfidenceCount int                    `json:"high_confidence_count"`
}
