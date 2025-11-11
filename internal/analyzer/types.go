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

// Severity levels for detections
const (
	SeverityNone     = "NONE"
	SeverityLow      = "LOW"
	SeverityMedium   = "MEDIUM"
	SeverityHigh     = "HIGH"
	SeverityCritical = "CRITICAL"
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

// ==================== ENHANCED DIAGNOSTIC TYPES ====================

// EnhancedDiagnosticData contains all the rich diagnostic information
type EnhancedDiagnosticData struct {
	ExecutiveSummary   *ExecutiveSummary         `json:"executive_summary"`
	DetailedRootCause  *DetailedRootCause        `json:"detailed_root_cause"`
	Timeline           *DiagnosticTimeline       `json:"timeline"`
	EnhancedActions    []*EnhancedActuatorAction `json:"enhanced_actions"`
	HealthIntelligence *HealthIntelligence       `json:"health_intelligence"`
	SLACompliance      *SLACompliance            `json:"sla_compliance"`
	MetricIntelligence *MetricIntelligence       `json:"metric_intelligence"`
	ImpactAnalysis     *ImpactAnalysis           `json:"impact_analysis"`
}

type ExecutiveSummary struct {
	OneLiner           string   `json:"one_liner"`
	SeverityLevel      string   `json:"severity_level"` // SEV-0 to SEV-4
	IncidentType       string   `json:"incident_type"`  // OUTAGE, DEGRADATION, ANOMALY
	KeyFindings        []string `json:"key_findings"`
	RequiresEscalation bool     `json:"requires_escalation"`
	EscalationLevel    string   `json:"escalation_level,omitempty"`
	EstimatedDowntime  string   `json:"estimated_downtime,omitempty"`
	RecoveryTime       string   `json:"recovery_time"`
	BusinessImpact     string   `json:"business_impact"`
}

type DetailedRootCause struct {
	PrimaryIssue        string                `json:"primary_issue"`
	Confidence          float64               `json:"confidence"`
	TriggerEvent        *TriggerEvent         `json:"trigger_event"`
	EvidenceChain       []*Evidence           `json:"evidence_chain"`
	PropagationPath     []string              `json:"propagation_path"`
	BlastRadius         *BlastRadius          `json:"blast_radius"`
	ContributingFactors []*ContributingFactor `json:"contributing_factors"`
	TimeToImpact        string                `json:"time_to_impact"`
	AffectedComponents  []string              `json:"affected_components"`
}

type TriggerEvent struct {
	Type        string                 `json:"type"`
	Timestamp   time.Time              `json:"timestamp"`
	Description string                 `json:"description"`
	Source      string                 `json:"source,omitempty"`
	Confidence  float64                `json:"confidence"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

type Evidence struct {
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Metric      string                 `json:"metric,omitempty"`
	Value       interface{}            `json:"value,omitempty"`
	Threshold   interface{}            `json:"threshold,omitempty"`
	Severity    string                 `json:"severity"`
	Timestamp   time.Time              `json:"timestamp"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

type BlastRadius struct {
	Scope            string   `json:"scope"`
	AffectedServices []string `json:"affected_services"`
	AffectedUsers    string   `json:"affected_users"`
	EstimatedReach   int      `json:"estimated_reach"`
	DownstreamImpact []string `json:"downstream_impact"`
	UpstreamImpact   []string `json:"upstream_impact"`
}

type ContributingFactor struct {
	Type         string  `json:"type"`
	Description  string  `json:"description"`
	Confidence   float64 `json:"confidence"`
	Relationship string  `json:"relationship"`
}

type DiagnosticTimeline struct {
	StartTime        time.Time         `json:"start_time"`
	DetectionTime    time.Time         `json:"detection_time"`
	Events           []*TimelineEvent  `json:"events"`
	KeyMilestones    []string          `json:"key_milestones"`
	PredictionWindow *PredictionWindow `json:"prediction_window"`
}

type TimelineEvent struct {
	Timestamp   time.Time              `json:"timestamp"`
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Severity    string                 `json:"severity"`
	Metrics     map[string]interface{} `json:"metrics,omitempty"`
}

type PredictionWindow struct {
	Next1Hour       *Prediction `json:"next_1_hour,omitempty"`
	Next6Hours      *Prediction `json:"next_6_hours,omitempty"`
	Next24Hours     *Prediction `json:"next_24_hours,omitempty"`
	ConfidenceLevel float64     `json:"confidence_level"`
}

type Prediction struct {
	Metric             string     `json:"metric"`
	CurrentValue       float64    `json:"current_value"`
	PredictedValue     float64    `json:"predicted_value"`
	ConfidenceInterval [2]float64 `json:"confidence_interval"`
	Trend              string     `json:"trend"`
	Likelihood         float64    `json:"likelihood"`
	RecommendedAction  string     `json:"recommended_action,omitempty"`
}

type EnhancedActuatorAction struct {
	ActionType   string      `json:"action_type"`
	Priority     string      `json:"priority"`
	TargetMetric string      `json:"target_metric"`
	CurrentValue interface{} `json:"current_value"`
	TargetValue  interface{} `json:"target_value"`
	Reason       string      `json:"reason"`
	Confidence   float64     `json:"confidence"`

	PreConditions   []string               `json:"pre_conditions"`
	PostConditions  []string               `json:"post_conditions"`
	SuccessCriteria []*SuccessCriterion    `json:"success_criteria"`
	RollbackPlan    *RollbackPlan          `json:"rollback_plan"`
	EstimatedImpact *ActionImpact          `json:"estimated_impact"`
	TimeWindow      *TimeWindow            `json:"time_window"`
	Dependencies    []string               `json:"dependencies,omitempty"`
	Parameters      map[string]interface{} `json:"parameters,omitempty"`
}

type SuccessCriterion struct {
	Metric    string  `json:"metric"`
	Operator  string  `json:"operator"`
	Threshold float64 `json:"threshold"`
	Duration  string  `json:"duration"`
	Priority  string  `json:"priority"`
}

type RollbackPlan struct {
	CanRollback      bool     `json:"can_rollback"`
	RollbackAction   string   `json:"rollback_action,omitempty"`
	RollbackSteps    []string `json:"rollback_steps,omitempty"`
	RollbackWindow   string   `json:"rollback_window,omitempty"`
	AutoRollback     bool     `json:"auto_rollback"`
	RollbackTriggers []string `json:"rollback_triggers,omitempty"`
}

type ActionImpact struct {
	UserImpact         string  `json:"user_impact"`
	AvailabilityImpact string  `json:"availability_impact"`
	PerformanceImpact  string  `json:"performance_impact"`
	CostImpact         float64 `json:"cost_impact,omitempty"`
	Duration           string  `json:"duration"`
	Reversible         bool    `json:"reversible"`
}

type TimeWindow struct {
	Earliest  time.Time `json:"earliest,omitempty"`
	Latest    time.Time `json:"latest,omitempty"`
	Preferred time.Time `json:"preferred,omitempty"`
	Urgency   string    `json:"urgency"`
	CanDelay  bool      `json:"can_delay"`
	MaxDelay  string    `json:"max_delay,omitempty"`
}

type HealthIntelligence struct {
	CurrentHealth   float64        `json:"current_health"`
	HealthHistory   *HealthHistory `json:"health_history"`
	HealthTrend     string         `json:"health_trend"`
	SystemStress    float64        `json:"system_stress"`
	StabilityIndex  float64        `json:"stability_index"`
	Predictability  float64        `json:"predictability"`
	AnomalyScore    float64        `json:"anomaly_score"`
	DegradationRate float64        `json:"degradation_rate"`
}

type HealthHistory struct {
	Last5Minutes  float64 `json:"last_5_minutes"`
	Last15Minutes float64 `json:"last_15_minutes"`
	Last30Minutes float64 `json:"last_30_minutes"`
	Last1Hour     float64 `json:"last_1_hour"`
}

type SLACompliance struct {
	OverallStatus     string                `json:"overall_status"`
	TimeToBreach      string                `json:"time_to_breach,omitempty"`
	BreachProbability float64               `json:"breach_probability"`
	Metrics           map[string]*SLAMetric `json:"metrics"`
	ViolationCount    int                   `json:"violation_count"`
	WarningCount      int                   `json:"warning_count"`
}

type SLAMetric struct {
	Name    string  `json:"name"`
	Target  float64 `json:"target"`
	Current float64 `json:"current"`
	Status  string  `json:"status"`
	Margin  float64 `json:"margin"`
	Trend   string  `json:"trend"`
}

type MetricIntelligence struct {
	Correlations      []*MetricCorrelation `json:"correlations"`
	AnomalousMetrics  []string             `json:"anomalous_metrics"`
	TrendingMetrics   []*TrendingMetric    `json:"trending_metrics"`
	ThresholdBreaches []*ThresholdBreach   `json:"threshold_breaches"`
}

type MetricCorrelation struct {
	Metric1      string  `json:"metric1"`
	Metric2      string  `json:"metric2"`
	Correlation  float64 `json:"correlation"`
	Strength     string  `json:"strength"`
	Causality    string  `json:"causality,omitempty"`
	Explanation  string  `json:"explanation"`
	Significance float64 `json:"significance"`
}

type TrendingMetric struct {
	Metric    string  `json:"metric"`
	Current   float64 `json:"current"`
	Trend     float64 `json:"trend"`
	Direction string  `json:"direction"`
	Concern   string  `json:"concern"`
}

type ThresholdBreach struct {
	Metric    string    `json:"metric"`
	Threshold float64   `json:"threshold"`
	Current   float64   `json:"current"`
	Severity  string    `json:"severity"`
	Timestamp time.Time `json:"timestamp"`
}

type ImpactAnalysis struct {
	UserImpact         string  `json:"user_impact"`
	AffectedUsersPct   string  `json:"affected_users_pct"`
	AvailabilityImpact string  `json:"availability_impact"`
	PerformanceScore   float64 `json:"performance_score"`
	BusinessImpact     string  `json:"business_impact"`
	RevenueImpact      string  `json:"revenue_impact,omitempty"`
	CostImpact         string  `json:"cost_impact"`
	RecoveryDifficulty string  `json:"recovery_difficulty"`
	EstimatedDowntime  string  `json:"estimated_downtime,omitempty"`
}
