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


type AdvancedDiagnosis struct {
	BasicDiagnosis *Diagnosis           `json:"basic_diagnosis"`
	RootCause      string               `json:"root_cause"`
	ImpactScore    float64              `json:"impact_score"`   // 0-100 score indicating severity and scope
	TrendAnalysis  map[string]string    `json:"trend_analysis"` // metric -> trend direction
	Correlations   []CorrelationInsight `json:"correlations"`   // Cross-detector correlations
	PriorityScore  float64              `json:"priority_score"` // 0-100 urgency score for triage
}

type CorrelationInsight struct {
	Detector1   string  `json:"detector1"`
	Detector2   string  `json:"detector2"`
	Correlation float64 `json:"correlation"` // 0-1 strength
	Explanation string  `json:"explanation"`
	Causality   string  `json:"causality"` // e.g., "A → B"
}

type ServiceComparison struct {
	ServiceName       string  `json:"service_name"`
	HealthScore       float64 `json:"health_score"` // 0-100, higher is better
	PrimaryIssue      string  `json:"primary_issue"`
	IssueCount        int     `json:"issue_count"`
	Severity          string  `json:"severity"`
	RequiresAttention bool    `json:"requires_attention"` // true if health < 80
}
