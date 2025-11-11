package analyzer

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/namansh70747/AURA-Autonomous-Unified-Reliability-Automation-Platform/internal/storage"
	"github.com/namansh70747/AURA-Autonomous-Unified-Reliability-Automation-Platform/pkg/logger"
	"go.uber.org/zap"
)

// UltimateAnalyzer integrates all AI-level components
type UltimateAnalyzer struct {
	featureExtractor *FeatureExtractor
	enhancedDetector *EnhancedDetector
	db               *storage.PostgresClient
}

func NewUltimateAnalyzer(db *storage.PostgresClient) *UltimateAnalyzer {
	fe := NewFeatureExtractor(db)
	ed := NewEnhancedDetector(fe)

	return &UltimateAnalyzer{
		featureExtractor: fe,
		enhancedDetector: ed,
		db:               db,
	}
}

// ActuatorAction represents a concrete action for the actuator
type ActuatorAction struct {
	ActionType   string                 `json:"action_type"`   // SCALE_UP, SCALE_DOWN, ROLLBACK, RESTART, ALERT, MONITOR
	Priority     string                 `json:"priority"`      // IMMEDIATE, HIGH, MEDIUM, LOW
	TargetMetric string                 `json:"target_metric"` // cpu, memory, replicas, etc.
	CurrentValue interface{}            `json:"current_value"`
	TargetValue  interface{}            `json:"target_value"`
	Reason       string                 `json:"reason"`
	Confidence   float64                `json:"confidence"`
	Parameters   map[string]interface{} `json:"parameters,omitempty"`
}

// RootCauseAnalysis represents the identified root cause
type RootCauseAnalysis struct {
	PrimaryIssue       string   `json:"primary_issue"`
	ContributingIssues []string `json:"contributing_issues"`
	Confidence         float64  `json:"confidence"`
	TimeToImpact       string   `json:"time_to_impact"`
	AffectedMetrics    []string `json:"affected_metrics"`
}

// UltimateDiagnosis represents comprehensive AI-level diagnosis
type UltimateDiagnosis struct {
	ServiceName      string
	Timestamp        time.Time
	AnalysisDuration time.Duration

	// Extracted features
	Features *ServiceFeatures

	// Primary detection (highest confidence)
	PrimaryDetection *Detection

	// All detections
	AllDetections []*Detection

	// Composite metrics
	HealthScore         float64 // 0-100
	StabilityIndex      float64 // 0-10
	PredictabilityScore float64 // 0-100
	SystemStress        float64 // 0-100

	// Decision support
	RiskLevel          string // LOW, NORMAL, MEDIUM, HIGH, CRITICAL
	ActionRequired     bool
	PredictiveInsights []string
	Recommendation     string

	// Actuator-ready outputs
	RootCause        *RootCauseAnalysis     `json:"root_cause"`
	ActuatorActions  []*ActuatorAction      `json:"actuator_actions"`
	ImpactAssessment map[string]interface{} `json:"impact_assessment"`

	// Traceability
	PredictionID string

	// ✨ ENHANCED DIAGNOSTIC DATA ✨
	EnhancedData *EnhancedDiagnosticData `json:"enhanced_data,omitempty"`
}

// DiagnoseService performs ultimate comprehensive diagnosis
func (ua *UltimateAnalyzer) DiagnoseService(ctx context.Context, serviceName string) (*UltimateDiagnosis, error) {
	startTime := time.Now()

	logger.Info("🔍 Starting AI-level diagnosis",
		zap.String("service", serviceName),
	)

	diagnosis := &UltimateDiagnosis{
		ServiceName:  serviceName,
		Timestamp:    time.Now(),
		PredictionID: uuid.New().String(),
	}

	// Step 1: Extract comprehensive features
	features, err := ua.featureExtractor.ExtractFeatures(ctx, serviceName, 30*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("feature extraction failed: %w", err)
	}
	diagnosis.Features = features

	// Step 2: Run all enhanced detectors
	detections := make([]*Detection, 0, 5)

	// Memory leak detection
	if d, err := ua.enhancedDetector.DetectMemoryLeakEnhanced(ctx, serviceName); err == nil {
		detections = append(detections, d)
	}

	// Resource exhaustion detection
	if d, err := ua.enhancedDetector.DetectResourceExhaustionEnhanced(ctx, serviceName); err == nil {
		detections = append(detections, d)
	}

	// Deployment bug detection
	if d, err := ua.enhancedDetector.DetectDeploymentBugEnhanced(ctx, serviceName); err == nil {
		detections = append(detections, d)
	}

	// External failure detection
	if d, err := ua.enhancedDetector.DetectExternalFailureEnhanced(ctx, serviceName); err == nil {
		detections = append(detections, d)
	}

	// Cascade failure detection
	if d, err := ua.enhancedDetector.DetectCascadeFailureEnhanced(ctx, serviceName); err == nil {
		detections = append(detections, d)
	}

	diagnosis.AllDetections = detections

	// Step 3: Determine primary detection (highest confidence among detected issues)
	var primaryDetection *Detection
	maxConfidence := 0.0

	for _, d := range detections {
		if d.Detected && d.Confidence > maxConfidence {
			maxConfidence = d.Confidence
			primaryDetection = d
		}
	}

	if primaryDetection == nil {
		// No issues detected - create healthy detection
		primaryDetection = &Detection{
			Type:           DetectionHealthy,
			ServiceName:    serviceName,
			Detected:       false,
			Confidence:     features.HealthScore,
			Severity:       SeverityNone,
			Evidence:       map[string]interface{}{"health_score": features.HealthScore},
			Recommendation: "✅ Service is operating normally. No action required.",
			Timestamp:      time.Now(),
		}
	}

	diagnosis.PrimaryDetection = primaryDetection

	// Step 4: Calculate composite scores (from features)
	diagnosis.HealthScore = features.HealthScore
	diagnosis.StabilityIndex = features.StabilityIndex
	diagnosis.PredictabilityScore = features.PredictabilityScore
	diagnosis.SystemStress = features.SystemStress

	// Step 5: Determine risk level
	diagnosis.RiskLevel = ua.determineRiskLevel(diagnosis)
	diagnosis.ActionRequired = diagnosis.RiskLevel == "CRITICAL" || diagnosis.RiskLevel == "HIGH"

	// Step 6: Generate predictive insights
	diagnosis.PredictiveInsights = ua.generatePredictiveInsights(features, detections)

	// Step 7: Generate root cause analysis
	diagnosis.RootCause = ua.analyzeRootCause(diagnosis)

	// Step 8: Generate actuator actions
	diagnosis.ActuatorActions = ua.generateActuatorActions(diagnosis)

	// Step 9: Generate impact assessment
	diagnosis.ImpactAssessment = ua.assessImpact(diagnosis)

	// Step 10: Generate actionable recommendation
	diagnosis.Recommendation = ua.generateRecommendation(diagnosis)

	// Step 11: 🌟 Generate Enhanced Diagnostic Data 🌟
	diagnosis.EnhancedData = ua.generateEnhancedData(diagnosis)

	diagnosis.AnalysisDuration = time.Since(startTime)

	logger.Info("✅ AI-level diagnosis complete",
		zap.String("service", serviceName),
		zap.String("primary_problem", string(primaryDetection.Type)),
		zap.Float64("confidence", primaryDetection.Confidence),
		zap.String("risk_level", diagnosis.RiskLevel),
		zap.Duration("duration", diagnosis.AnalysisDuration),
	)

	return diagnosis, nil
}

func (ua *UltimateAnalyzer) determineRiskLevel(diag *UltimateDiagnosis) string {
	// CRITICAL: Severity is critical OR health score < 30
	if diag.PrimaryDetection.Severity == SeverityCritical || diag.HealthScore < 30 {
		return "CRITICAL"
	}

	// HIGH: Severity is high OR health score < 50 OR system stress > 80
	if diag.PrimaryDetection.Severity == SeverityHigh || diag.HealthScore < 50 || diag.SystemStress > 80 {
		return "HIGH"
	}

	// MEDIUM: Severity is medium OR health score < 70
	if diag.PrimaryDetection.Severity == SeverityMedium || diag.HealthScore < 70 {
		return "MEDIUM"
	}

	// LOW: Any detection with low severity
	if diag.PrimaryDetection.Severity == SeverityLow {
		return "LOW"
	}

	// NORMAL: No issues
	return "NORMAL"
}

func (ua *UltimateAnalyzer) generatePredictiveInsights(features *ServiceFeatures, detections []*Detection) []string {
	insights := make([]string, 0)

	// Trend-based predictions
	if features.MemoryTrend > 0.5 {
		minutesToFull := (100 - features.MemoryMean) / features.MemoryTrend
		if minutesToFull > 0 && minutesToFull < 120 {
			insights = append(insights, fmt.Sprintf("📈 Memory increasing at %.2f%%/min - potential exhaustion in %.0f minutes",
				features.MemoryTrend, minutesToFull))
		}
	}

	if features.CPUTrend > 0.3 {
		insights = append(insights, fmt.Sprintf("📈 CPU trending upward (%.2f%%/min) - monitor for sustained growth",
			features.CPUTrend))
	}

	// Pattern-based predictions
	if features.HasPeriodicPattern {
		insights = append(insights, fmt.Sprintf("🔄 Periodic pattern detected (%.0fs cycle) - behavior is predictable",
			features.PeriodLength.Seconds()))
	}

	// Correlation warnings
	if features.CPUErrorCorr > 0.7 {
		insights = append(insights, "⚠️ Strong CPU-Error correlation - errors likely caused by resource saturation")
	}

	if features.LatencyErrorCorr > 0.6 {
		insights = append(insights, "⚠️ Latency-Error correlation - external dependency issues suspected")
	}

	// Stability warnings
	if features.StabilityIndex < 4 {
		insights = append(insights, fmt.Sprintf("⚡ Low stability (%.1f/10) - system is volatile and unpredictable",
			features.StabilityIndex))
	}

	// Multiple issue warning
	detectedCount := 0
	for _, d := range detections {
		if d.Detected {
			detectedCount++
		}
	}
	if detectedCount > 1 {
		insights = append(insights, fmt.Sprintf("🚨 %d concurrent issues detected - cascade failure risk", detectedCount))
	}

	if len(insights) == 0 {
		insights = append(insights, "✅ System metrics stable - no immediate concerns")
	}

	return insights
}

func (ua *UltimateAnalyzer) generateRecommendation(diag *UltimateDiagnosis) string {
	var recommendation string

	// Build urgent action header based on risk level
	switch diag.RiskLevel {
	case "CRITICAL":
		recommendation = "🚨 **CRITICAL ACTION REQUIRED**\n\n"
		recommendation += fmt.Sprintf("**Primary Issue:** %s (%.1f%% confidence)\n", diag.PrimaryDetection.Type, diag.PrimaryDetection.Confidence)
		recommendation += fmt.Sprintf("**Time to Impact:** %s\n\n", diag.RootCause.TimeToImpact)
	case "HIGH":
		recommendation = "⚠️ **URGENT ACTION REQUIRED**\n\n"
		recommendation += fmt.Sprintf("**Primary Issue:** %s (%.1f%% confidence)\n", diag.PrimaryDetection.Type, diag.PrimaryDetection.Confidence)
		recommendation += fmt.Sprintf("**Time to Impact:** %s\n\n", diag.RootCause.TimeToImpact)
	case "MEDIUM":
		recommendation = "⚡ **ATTENTION REQUIRED**\n\n"
		recommendation += fmt.Sprintf("**Detected Issue:** %s (%.1f%% confidence)\n\n", diag.PrimaryDetection.Type, diag.PrimaryDetection.Confidence)
	case "LOW":
		recommendation = "📊 **ADVISORY NOTICE**\n\n"
		recommendation += fmt.Sprintf("**Monitoring:** %s (%.1f%% confidence)\n\n", diag.PrimaryDetection.Type, diag.PrimaryDetection.Confidence)
	default:
		recommendation = "✅ **SYSTEM HEALTHY**\n\n"
		recommendation += fmt.Sprintf("No critical issues detected. Health Score: %.0f/100\n\n", diag.HealthScore)
	}

	// Add immediate actions
	if len(diag.ActuatorActions) > 0 {
		recommendation += "**IMMEDIATE ACTIONS:**\n"
		immediateCount := 0
		for _, action := range diag.ActuatorActions {
			if action.Priority == "IMMEDIATE" {
				immediateCount++
				recommendation += fmt.Sprintf("%d. **%s**: %s\n   - Current: %v → Target: %v\n   - Reason: %s\n",
					immediateCount, action.ActionType, action.TargetMetric,
					action.CurrentValue, action.TargetValue, action.Reason)
			}
		}

		// Add high priority actions
		highCount := 0
		for _, action := range diag.ActuatorActions {
			if action.Priority == "HIGH" {
				if highCount == 0 {
					recommendation += "\n**HIGH PRIORITY ACTIONS:**\n"
				}
				highCount++
				recommendation += fmt.Sprintf("%d. **%s**: %s\n",
					highCount, action.ActionType, action.Reason)
			}
		}
		recommendation += "\n"
	}

	// Add root cause explanation
	if len(diag.RootCause.ContributingIssues) > 0 {
		recommendation += "**ROOT CAUSE ANALYSIS:**\n"
		recommendation += fmt.Sprintf("- Primary Issue: %s\n", diag.RootCause.PrimaryIssue)
		recommendation += "- Contributing Factors:\n"
		for _, issue := range diag.RootCause.ContributingIssues {
			recommendation += fmt.Sprintf("  • %s\n", issue)
		}
		recommendation += "\n"
	}

	// Add affected metrics with severity
	if len(diag.RootCause.AffectedMetrics) > 0 {
		recommendation += "**AFFECTED METRICS:**\n"
		for _, metric := range diag.RootCause.AffectedMetrics {
			recommendation += fmt.Sprintf("- %s\n", metric)
		}
		recommendation += "\n"
	}

	// Add predictive insights
	if len(diag.PredictiveInsights) > 0 {
		recommendation += "**PREDICTIVE INSIGHTS:**\n"
		for _, insight := range diag.PredictiveInsights {
			recommendation += fmt.Sprintf("• %s\n", insight)
		}
		recommendation += "\n"
	}

	// Add health assessment
	recommendation += "**HEALTH ASSESSMENT:**\n"
	recommendation += fmt.Sprintf("- Overall Health: %.0f/100 ", diag.HealthScore)
	if diag.HealthScore < 30 {
		recommendation += "(CRITICAL)\n"
	} else if diag.HealthScore < 50 {
		recommendation += "(POOR)\n"
	} else if diag.HealthScore < 70 {
		recommendation += "(FAIR)\n"
	} else if diag.HealthScore < 90 {
		recommendation += "(GOOD)\n"
	} else {
		recommendation += "(EXCELLENT)\n"
	}
	recommendation += fmt.Sprintf("- System Stress: %.0f/100\n", diag.SystemStress)
	recommendation += fmt.Sprintf("- Stability Index: %.1f/10\n", diag.StabilityIndex)
	recommendation += fmt.Sprintf("- Predictability: %.0f/100\n", diag.PredictabilityScore)
	recommendation += "\n"

	// Add contextual recommendations based on issue type
	recommendation += "**NEXT STEPS:**\n"
	switch diag.PrimaryDetection.Type {
	case DetectionResourceExhaustion:
		recommendation += "1. Execute scaling actions immediately\n"
		recommendation += "2. Monitor resource utilization post-scaling\n"
		recommendation += "3. Investigate root cause of resource spike\n"
		recommendation += "4. Consider implementing auto-scaling policies\n"
	case DetectionMemoryLeak:
		recommendation += "1. Execute rolling restart to reclaim memory\n"
		recommendation += "2. Capture heap dump for analysis\n"
		recommendation += "3. Review recent code changes for memory allocation patterns\n"
		recommendation += "4. Implement memory profiling in staging environment\n"
	case DetectionDeploymentBug:
		recommendation += "1. Execute rollback immediately\n"
		recommendation += "2. Verify error rate reduction post-rollback\n"
		recommendation += "3. Analyze error logs to identify bug\n"
		recommendation += "4. Fix and test in staging before redeployment\n"
	case DetectionCascadingFailure:
		recommendation += "1. Enable circuit breaker to prevent cascade\n"
		recommendation += "2. Scale up affected services\n"
		recommendation += "3. Identify and isolate root cause service\n"
		recommendation += "4. Implement bulkhead pattern for isolation\n"
	case DetectionExternalFailure:
		recommendation += "1. Enable fallback/cache mechanisms\n"
		recommendation += "2. Implement retry with exponential backoff\n"
		recommendation += "3. Contact external service provider\n"
		recommendation += "4. Review SLA and failover strategies\n"
	default:
		if diag.HealthScore < 80 {
			recommendation += "1. Continue monitoring key metrics\n"
			recommendation += "2. Review recent changes and deployments\n"
			recommendation += "3. Verify alert thresholds are appropriate\n"
		} else {
			recommendation += "1. Maintain current monitoring\n"
			recommendation += "2. No immediate action required\n"
		}
	}

	recommendation += "\n"
	recommendation += fmt.Sprintf("**Diagnosis ID:** %s\n", diag.PredictionID)
	recommendation += fmt.Sprintf("**Generated:** %s\n", diag.Timestamp.Format(time.RFC3339))

	return recommendation
}

// analyzeRootCause performs deep root cause analysis with evidence
func (ua *UltimateAnalyzer) analyzeRootCause(diag *UltimateDiagnosis) *RootCauseAnalysis {
	features := diag.Features

	rca := &RootCauseAnalysis{
		PrimaryIssue:       string(diag.PrimaryDetection.Type),
		ContributingIssues: make([]string, 0),
		Confidence:         diag.PrimaryDetection.Confidence,
		AffectedMetrics:    make([]string, 0),
	}

	// Identify contributing issues with detailed analysis
	for _, d := range diag.AllDetections {
		if d.Detected && d.Type != diag.PrimaryDetection.Type {
			// Add relationship context
			relationship := ua.determineIssueRelationship(diag.PrimaryDetection.Type, d.Type)
			rca.ContributingIssues = append(rca.ContributingIssues,
				fmt.Sprintf("%s (%.1f%% confidence) - %s", d.Type, d.Confidence, relationship))
		}
	}

	// Advanced time-to-impact calculation with multiple scenarios
	rca.TimeToImpact = ua.calculateTimeToImpact(diag, features)

	// Detailed affected metrics with severity
	if features.CPUMean > 90 {
		rca.AffectedMetrics = append(rca.AffectedMetrics, fmt.Sprintf("cpu (CRITICAL: %.1f%%)", features.CPUMean))
	} else if features.CPUMean > 80 {
		rca.AffectedMetrics = append(rca.AffectedMetrics, fmt.Sprintf("cpu (HIGH: %.1f%%)", features.CPUMean))
	} else if features.CPUMean > 70 || features.CPUTrend > 0.5 {
		rca.AffectedMetrics = append(rca.AffectedMetrics, fmt.Sprintf("cpu (ELEVATED: %.1f%%)", features.CPUMean))
	}

	if features.MemoryMean > 90 {
		rca.AffectedMetrics = append(rca.AffectedMetrics, fmt.Sprintf("memory (CRITICAL: %.1f%%)", features.MemoryMean))
	} else if features.MemoryMean > 80 {
		rca.AffectedMetrics = append(rca.AffectedMetrics, fmt.Sprintf("memory (HIGH: %.1f%%)", features.MemoryMean))
	} else if features.MemoryMean > 70 || features.MemoryTrend > 0.5 {
		rca.AffectedMetrics = append(rca.AffectedMetrics, fmt.Sprintf("memory (ELEVATED: %.1f%%)", features.MemoryMean))
	}

	if features.ErrorRateMean > 50 {
		rca.AffectedMetrics = append(rca.AffectedMetrics, fmt.Sprintf("errors (CRITICAL: %.1f/min)", features.ErrorRateMean))
	} else if features.ErrorRateMean > 20 {
		rca.AffectedMetrics = append(rca.AffectedMetrics, fmt.Sprintf("errors (HIGH: %.1f/min)", features.ErrorRateMean))
	} else if features.ErrorRateMean > 5 {
		rca.AffectedMetrics = append(rca.AffectedMetrics, fmt.Sprintf("errors (ELEVATED: %.1f/min)", features.ErrorRateMean))
	}

	if features.LatencyP95 > 2000 {
		rca.AffectedMetrics = append(rca.AffectedMetrics, fmt.Sprintf("latency (CRITICAL: %.0fms p95)", features.LatencyP95))
	} else if features.LatencyP95 > 1000 {
		rca.AffectedMetrics = append(rca.AffectedMetrics, fmt.Sprintf("latency (HIGH: %.0fms p95)", features.LatencyP95))
	} else if features.LatencyP95 > 500 {
		rca.AffectedMetrics = append(rca.AffectedMetrics, fmt.Sprintf("latency (ELEVATED: %.0fms p95)", features.LatencyP95))
	}

	return rca
}

// determineIssueRelationship explains how two issues relate to each other
func (ua *UltimateAnalyzer) determineIssueRelationship(primary, secondary DetectionType) string {
	relationships := map[string]map[DetectionType]string{
		string(DetectionResourceExhaustion): {
			DetectionMemoryLeak:       "likely caused by memory leak",
			DetectionDeploymentBug:    "may be triggered by deployment",
			DetectionCascadingFailure: "causing cascade effect",
			DetectionExternalFailure:  "external pressure adding to exhaustion",
		},
		string(DetectionMemoryLeak): {
			DetectionResourceExhaustion: "leading to resource exhaustion",
			DetectionCascadingFailure:   "triggering cascade failure",
		},
		string(DetectionDeploymentBug): {
			DetectionResourceExhaustion: "causing resource spike",
			DetectionCascadingFailure:   "triggering system-wide issues",
			DetectionExternalFailure:    "breaking external dependencies",
		},
		string(DetectionCascadingFailure): {
			DetectionResourceExhaustion: "multiple resource exhaustion",
			DetectionMemoryLeak:         "progressive memory degradation",
			DetectionExternalFailure:    "upstream failures propagating",
		},
		string(DetectionExternalFailure): {
			DetectionCascadingFailure:   "external failures cascading internally",
			DetectionResourceExhaustion: "retry storms exhausting resources",
		},
	}

	if primaryRels, ok := relationships[string(primary)]; ok {
		if rel, ok := primaryRels[secondary]; ok {
			return rel
		}
	}

	return "may be related"
}

// calculateTimeToImpact provides detailed time-to-impact analysis
func (ua *UltimateAnalyzer) calculateTimeToImpact(diag *UltimateDiagnosis, features *ServiceFeatures) string {
	// Already critical
	if diag.RiskLevel == "CRITICAL" && diag.HealthScore < 30 {
		return "⚠️ IMMEDIATE - Service already in critical state, action required NOW"
	}

	// Memory exhaustion prediction
	if features.MemoryTrend > 0.5 {
		minutesToFull := (100 - features.MemoryMean) / features.MemoryTrend
		if minutesToFull > 0 && minutesToFull < 5 {
			return fmt.Sprintf("⚠️ IMMEDIATE - Memory exhaustion in %.0f minutes", minutesToFull)
		} else if minutesToFull > 0 && minutesToFull < 15 {
			return fmt.Sprintf("🔴 CRITICAL - Memory exhaustion in %.0f minutes", minutesToFull)
		} else if minutesToFull > 0 && minutesToFull < 60 {
			return fmt.Sprintf("🟠 HIGH - Memory exhaustion in %.0f minutes", minutesToFull)
		}
	}

	// CPU exhaustion prediction
	if features.CPUTrend > 1.0 {
		minutesToFull := (100 - features.CPUMean) / features.CPUTrend
		if minutesToFull > 0 && minutesToFull < 5 {
			return fmt.Sprintf("⚠️ IMMEDIATE - CPU exhaustion in %.0f minutes", minutesToFull)
		} else if minutesToFull > 0 && minutesToFull < 15 {
			return fmt.Sprintf("🔴 CRITICAL - CPU exhaustion in %.0f minutes", minutesToFull)
		} else if minutesToFull > 0 && minutesToFull < 60 {
			return fmt.Sprintf("🟠 HIGH - CPU exhaustion in %.0f minutes", minutesToFull)
		}
	}

	// Error rate explosion
	if features.ErrorRateTrend > 5 {
		return "🔴 CRITICAL - Error rate rapidly increasing, < 10 minutes to service failure"
	}

	// Based on risk level
	switch diag.RiskLevel {
	case "CRITICAL":
		return "🔴 CRITICAL - Immediate action required within 5 minutes"
	case "HIGH":
		return "🟠 HIGH - Action required within 15 minutes"
	case "MEDIUM":
		return "🟡 MEDIUM - Action recommended within 1 hour"
	case "LOW":
		return "🟢 LOW - Monitor over next 4 hours"
	default:
		return "✅ NORMAL - No immediate time pressure"
	}
}

// generateActuatorActions generates concrete actions for the actuator
func (ua *UltimateAnalyzer) generateActuatorActions(diag *UltimateDiagnosis) []*ActuatorAction {
	actions := make([]*ActuatorAction, 0)
	features := diag.Features

	// Priority mapping based on risk level
	priorityMap := map[string]string{
		"CRITICAL": "IMMEDIATE",
		"HIGH":     "HIGH",
		"MEDIUM":   "MEDIUM",
		"LOW":      "LOW",
		"NORMAL":   "LOW",
	}
	priority := priorityMap[diag.RiskLevel]

	// Generate actions based on detection type
	switch diag.PrimaryDetection.Type {
	case DetectionResourceExhaustion:
		// Check if CPU or Memory is the issue
		if features.CPUMean > 80 || features.CPUVolatility > 20 {
			// Calculate recommended replicas based on load
			currentLoad := features.CPUMean
			targetLoad := 60.0 // Target 60% utilization
			recommendedReplicas := int(math.Ceil(currentLoad / targetLoad))
			if recommendedReplicas < 2 {
				recommendedReplicas = 2
			}
			if recommendedReplicas > 10 {
				recommendedReplicas = 10 // Cap at 10
			}

			actions = append(actions, &ActuatorAction{
				ActionType:   "SCALE_UP",
				Priority:     priority,
				TargetMetric: "replicas",
				CurrentValue: 1,
				TargetValue:  recommendedReplicas,
				Reason:       fmt.Sprintf("CPU at %.1f%% (avg) with %.1f%% volatility - scale to %d replicas to achieve 60%% target utilization", features.CPUMean, features.CPUVolatility, recommendedReplicas),
				Confidence:   diag.PrimaryDetection.Confidence,
				Parameters: map[string]interface{}{
					"cpu_current":          features.CPUMean,
					"cpu_volatility":       features.CPUVolatility,
					"cpu_target":           targetLoad,
					"scale_increment":      recommendedReplicas - 1,
					"recommended_replicas": recommendedReplicas,
					"scaling_strategy":     "horizontal",
					"expected_cpu_after":   fmt.Sprintf("%.1f%%", currentLoad/float64(recommendedReplicas)),
				},
			})
		}

		if features.MemoryMean > 80 {
			// Calculate memory increase needed
			currentMemPct := features.MemoryMean
			var recommendedMemory string
			var increaseMultiplier float64

			if currentMemPct > 95 {
				recommendedMemory = "2Gi"
				increaseMultiplier = 4.0
			} else if currentMemPct > 90 {
				recommendedMemory = "1.5Gi"
				increaseMultiplier = 3.0
			} else {
				recommendedMemory = "1Gi"
				increaseMultiplier = 2.0
			}

			actions = append(actions, &ActuatorAction{
				ActionType:   "INCREASE_LIMITS",
				Priority:     priority,
				TargetMetric: "memory",
				CurrentValue: "512Mi",
				TargetValue:  recommendedMemory,
				Reason:       fmt.Sprintf("Memory at %.1f%% with %.1f%%/min growth rate - increase to %s (%.1fx) to prevent OOM kills", features.MemoryMean, features.MemoryTrend, recommendedMemory, increaseMultiplier),
				Confidence:   diag.PrimaryDetection.Confidence,
				Parameters: map[string]interface{}{
					"memory_current":        features.MemoryMean,
					"memory_trend":          features.MemoryTrend,
					"memory_threshold":      80.0,
					"recommended_increase":  fmt.Sprintf("%.1fx", increaseMultiplier),
					"expected_memory_after": fmt.Sprintf("%.1f%%", currentMemPct/increaseMultiplier),
					"oom_risk":              currentMemPct > 95,
				},
			})
		}

		// Add load balancing if needed
		if features.CPUVolatility > 30 {
			actions = append(actions, &ActuatorAction{
				ActionType:   "ENABLE_LOAD_BALANCER",
				Priority:     "HIGH",
				TargetMetric: "traffic_distribution",
				CurrentValue: "unbalanced",
				TargetValue:  "balanced",
				Reason:       fmt.Sprintf("CPU volatility at %.1f%% indicates uneven load distribution - enable intelligent load balancing", features.CPUVolatility),
				Confidence:   85.0,
				Parameters: map[string]interface{}{
					"algorithm":        "least_connections",
					"health_check":     "enabled",
					"session_affinity": false,
				},
			})
		}

	case DetectionDeploymentBug:
		// Calculate deployment version
		rollbackTarget := "previous_stable"
		rollbackReason := fmt.Sprintf("Error rate at %.1f%% with %.1fx spike intensity after deployment", features.ErrorRateMean, features.ErrorRateSpikiness)

		actions = append(actions, &ActuatorAction{
			ActionType:   "ROLLBACK",
			Priority:     priority,
			TargetMetric: "deployment",
			CurrentValue: "latest",
			TargetValue:  rollbackTarget,
			Reason:       rollbackReason,
			Confidence:   diag.PrimaryDetection.Confidence,
			Parameters: map[string]interface{}{
				"error_rate":          features.ErrorRateMean,
				"error_spikiness":     features.ErrorRateSpikiness,
				"stability":           features.StabilityIndex,
				"rollback_to":         rollbackTarget,
				"rollback_strategy":   "immediate",
				"verification_window": "5m",
				"auto_forward":        false, // Don't auto-deploy after rollback
			},
		})

		// Add post-rollback monitoring
		actions = append(actions, &ActuatorAction{
			ActionType:   "MONITOR",
			Priority:     "HIGH",
			TargetMetric: "errors",
			CurrentValue: features.ErrorRateMean,
			TargetValue:  5.0,
			Reason:       "Monitor error rate post-rollback to confirm recovery and detect any lingering issues",
			Confidence:   95.0,
			Parameters: map[string]interface{}{
				"monitor_duration":  "15m",
				"alert_threshold":   10.0,
				"sample_interval":   "30s",
				"success_threshold": 2.0,
				"metrics":           []string{"error_rate", "latency_p95", "success_rate"},
			},
		})

		// Add alert to engineering
		actions = append(actions, &ActuatorAction{
			ActionType:   "ALERT",
			Priority:     "HIGH",
			TargetMetric: "deployment",
			CurrentValue: "failed",
			TargetValue:  "investigated",
			Reason:       "Deployment introduced bugs - engineering investigation required before next deploy",
			Confidence:   diag.PrimaryDetection.Confidence,
			Parameters: map[string]interface{}{
				"alert_channel":  "engineering",
				"severity":       "high",
				"include_logs":   true,
				"error_samples":  10,
				"blocked_deploy": true,
			},
		})

	case DetectionMemoryLeak:
		// Immediate mitigation
		actions = append(actions, &ActuatorAction{
			ActionType:   "RESTART",
			Priority:     priority,
			TargetMetric: "pods",
			CurrentValue: "running",
			TargetValue:  "restarted",
			Reason:       fmt.Sprintf("Memory leak detected (%.2f%%/min growth, %.1f%% current) - rolling restart to reclaim memory", features.MemoryTrend, features.MemoryMean),
			Confidence:   diag.PrimaryDetection.Confidence,
			Parameters: map[string]interface{}{
				"restart_type":     "rolling",
				"memory_trend":     features.MemoryTrend,
				"current_memory":   features.MemoryMean,
				"max_surge":        1,
				"max_unavailable":  0,
				"grace_period":     "30s",
				"restart_interval": "2m",
			},
		})

		// Long-term fix
		actions = append(actions, &ActuatorAction{
			ActionType:   "ALERT",
			Priority:     "HIGH",
			TargetMetric: "memory",
			CurrentValue: features.MemoryMean,
			TargetValue:  60.0,
			Reason:       fmt.Sprintf("Memory leak (%.3f autocorrelation) requires code investigation - heap profiling recommended", features.MemoryAutocorrelation),
			Confidence:   diag.PrimaryDetection.Confidence,
			Parameters: map[string]interface{}{
				"alert_channel":          "engineering",
				"heap_dump":              true,
				"profile":                "memory",
				"profile_duration":       "5m",
				"include_goroutines":     true,
				"investigation_priority": "high",
			},
		})

	case DetectionCascadingFailure:
		// Immediate circuit breaker
		actions = append(actions, &ActuatorAction{
			ActionType:   "CIRCUIT_BREAKER",
			Priority:     "IMMEDIATE",
			TargetMetric: "traffic",
			CurrentValue: "100%",
			TargetValue:  "50%",
			Reason:       fmt.Sprintf("Cascading failure across %d metrics (health: %.0f/100) - enable circuit breaker to prevent total outage", len(diag.RootCause.AffectedMetrics), diag.HealthScore),
			Confidence:   diag.PrimaryDetection.Confidence,
			Parameters: map[string]interface{}{
				"breaker_threshold":  0.5,
				"timeout":            "30s",
				"recovery_time":      "5m",
				"half_open_requests": 10,
				"failure_threshold":  5,
			},
		})

		// Aggressive scaling
		actions = append(actions, &ActuatorAction{
			ActionType:   "SCALE_UP",
			Priority:     "IMMEDIATE",
			TargetMetric: "replicas",
			CurrentValue: 1,
			TargetValue:  5,
			Reason:       fmt.Sprintf("Multiple metrics degrading (%.0f/100 stress) - aggressive scaling to 5x capacity required", diag.SystemStress),
			Confidence:   diag.PrimaryDetection.Confidence,
			Parameters: map[string]interface{}{
				"scale_factor":    5,
				"urgency":         "high",
				"strategy":        "aggressive",
				"health_check":    "enabled",
				"readiness_delay": "15s",
			},
		})

	case DetectionExternalFailure:
		actions = append(actions, &ActuatorAction{
			ActionType:   "ENABLE_FALLBACK",
			Priority:     priority,
			TargetMetric: "dependencies",
			CurrentValue: "direct",
			TargetValue:  "cached",
			Reason:       fmt.Sprintf("External dependency failure (%.3f latency-error correlation) - enable cache/fallback to maintain service", features.LatencyErrorCorr),
			Confidence:   diag.PrimaryDetection.Confidence,
			Parameters: map[string]interface{}{
				"fallback_type":   "cache",
				"ttl":             "5m",
				"retry_strategy":  "exponential",
				"max_retries":     3,
				"circuit_breaker": true,
				"cache_warmup":    true,
			},
		})

		// Add retry with backoff
		actions = append(actions, &ActuatorAction{
			ActionType:   "CONFIGURE_RETRY",
			Priority:     "HIGH",
			TargetMetric: "external_calls",
			CurrentValue: "no_retry",
			TargetValue:  "exponential_backoff",
			Reason:       "Configure intelligent retry with exponential backoff to reduce external service pressure",
			Confidence:   90.0,
			Parameters: map[string]interface{}{
				"initial_delay": "100ms",
				"max_delay":     "10s",
				"multiplier":    2.0,
				"max_attempts":  3,
				"jitter":        true,
			},
		})
	}

	// Add monitoring action if health score is low but no critical actions
	if len(actions) == 0 && diag.HealthScore < 70 {
		actions = append(actions, &ActuatorAction{
			ActionType:   "MONITOR",
			Priority:     priority,
			TargetMetric: "health",
			CurrentValue: diag.HealthScore,
			TargetValue:  80.0,
			Reason:       "Health score degrading - increase monitoring frequency",
			Confidence:   80.0,
			Parameters: map[string]interface{}{
				"monitor_interval": "30s",
				"alert_threshold":  60.0,
			},
		})
	}

	return actions
}

// assessImpact assesses the impact of the current situation
func (ua *UltimateAnalyzer) assessImpact(diag *UltimateDiagnosis) map[string]interface{} {
	impact := make(map[string]interface{})
	features := diag.Features

	// Service availability impact
	availabilityImpact := "NONE"
	if diag.HealthScore < 30 {
		availabilityImpact = "CRITICAL - Service likely experiencing failures"
	} else if diag.HealthScore < 50 {
		availabilityImpact = "HIGH - Service degraded, user impact likely"
	} else if diag.HealthScore < 70 {
		availabilityImpact = "MEDIUM - Service performance reduced"
	} else if diag.HealthScore < 90 {
		availabilityImpact = "LOW - Minor degradation"
	}
	impact["availability"] = availabilityImpact

	// Performance impact
	performanceScore := 100.0
	if features.CPUMean > 80 {
		performanceScore -= 30
	} else if features.CPUMean > 60 {
		performanceScore -= 15
	}
	if features.MemoryMean > 80 {
		performanceScore -= 30
	} else if features.MemoryMean > 60 {
		performanceScore -= 15
	}
	if features.LatencyP95 > 1000 {
		performanceScore -= 25
	} else if features.LatencyP95 > 500 {
		performanceScore -= 10
	}
	impact["performance_score"] = fmt.Sprintf("%.0f/100", performanceScore)

	// User impact estimation
	userImpact := "NONE"
	affectedUsers := "0%"
	if features.ErrorRateMean > 50 {
		userImpact = "SEVERE"
		affectedUsers = "> 50%"
	} else if features.ErrorRateMean > 20 {
		userImpact = "HIGH"
		affectedUsers = "20-50%"
	} else if features.ErrorRateMean > 5 {
		userImpact = "MODERATE"
		affectedUsers = "5-20%"
	} else if features.ErrorRateMean > 1 {
		userImpact = "LOW"
		affectedUsers = "< 5%"
	}
	impact["user_impact"] = userImpact
	impact["estimated_affected_users"] = affectedUsers

	// Business impact
	businessImpact := "NONE"
	switch diag.RiskLevel {
	case "CRITICAL":
		businessImpact = "HIGH - Revenue and SLA impact likely"
	case "HIGH":
		businessImpact = "MEDIUM - Potential SLA violations"
	case "MEDIUM":
		businessImpact = "LOW - Monitor for SLA impact"
	}
	impact["business_impact"] = businessImpact

	// Cost impact (resource waste or scaling costs)
	costImpact := "NEUTRAL"
	if diag.SystemStress > 90 {
		costImpact = "SCALING_REQUIRED - Additional resources needed (increased cost)"
	} else if features.CPUMean < 20 && features.MemoryMean < 30 {
		costImpact = "OVER_PROVISIONED - Resources underutilized (waste)"
	}
	impact["cost_impact"] = costImpact

	// Recovery difficulty
	recoveryDifficulty := "EASY"
	switch diag.PrimaryDetection.Type {
	case DetectionMemoryLeak:
		recoveryDifficulty = "HARD - Requires code fix and deployment"
	case DetectionCascadingFailure:
		recoveryDifficulty = "HARD - Multiple systems affected"
	case DetectionResourceExhaustion:
		recoveryDifficulty = "MEDIUM - Requires scaling or optimization"
	case DetectionDeploymentBug:
		recoveryDifficulty = "EASY - Rollback available"
	}
	impact["recovery_difficulty"] = recoveryDifficulty

	return impact
}

// FeatureExtractor returns the feature extractor
func (ua *UltimateAnalyzer) FeatureExtractor() *FeatureExtractor {
	return ua.featureExtractor
}

// EnhancedDetector returns the enhanced detector
func (ua *UltimateAnalyzer) EnhancedDetector() *EnhancedDetector {
	return ua.enhancedDetector
}

// ================================================================================
// 🌟 ENHANCED DIAGNOSTIC DATA GENERATION 🌟
// ================================================================================

// generateEnhancedData creates comprehensive enhanced diagnostic data
func (ua *UltimateAnalyzer) generateEnhancedData(diag *UltimateDiagnosis) *EnhancedDiagnosticData {
	enhanced := &EnhancedDiagnosticData{}

	// 1. Executive Summary
	enhanced.ExecutiveSummary = ua.buildExecutiveSummary(diag)

	// 2. Detailed Root Cause
	enhanced.DetailedRootCause = ua.buildDetailedRootCause(diag)

	// 3. Timeline
	enhanced.Timeline = ua.buildTimeline(diag)

	// 4. Enhanced Actions
	enhanced.EnhancedActions = ua.buildEnhancedActions(diag)

	// 5. Health Intelligence
	enhanced.HealthIntelligence = ua.buildHealthIntelligence(diag)

	// 6. SLA Compliance
	enhanced.SLACompliance = ua.buildSLACompliance(diag)

	// 7. Metric Intelligence
	enhanced.MetricIntelligence = ua.buildMetricIntelligence(diag)

	// 8. Impact Analysis
	enhanced.ImpactAnalysis = ua.buildImpactAnalysis(diag)

	return enhanced
}

// buildExecutiveSummary creates C-level summary
func (ua *UltimateAnalyzer) buildExecutiveSummary(diag *UltimateDiagnosis) *ExecutiveSummary {
	summary := &ExecutiveSummary{
		KeyFindings: make([]string, 0),
	}

	// One-liner
	summary.OneLiner = fmt.Sprintf("%s detected in %s with %.1f%% confidence - %s risk",
		diag.PrimaryDetection.Type, diag.ServiceName, diag.PrimaryDetection.Confidence, diag.RiskLevel)

	// Severity classification
	switch diag.RiskLevel {
	case "CRITICAL":
		if diag.HealthScore < 30 {
			summary.SeverityLevel = "SEV-0" // Complete outage
			summary.IncidentType = "OUTAGE"
		} else {
			summary.SeverityLevel = "SEV-1" // Critical degradation
			summary.IncidentType = "DEGRADATION"
		}
		summary.RequiresEscalation = true
		summary.EscalationLevel = "MANAGEMENT"
	case "HIGH":
		summary.SeverityLevel = "SEV-2"
		summary.IncidentType = "DEGRADATION"
		summary.RequiresEscalation = true
		summary.EscalationLevel = "ENGINEERING"
	case "MEDIUM":
		summary.SeverityLevel = "SEV-3"
		summary.IncidentType = "DEGRADATION"
		summary.EscalationLevel = "ONCALL"
	default:
		summary.SeverityLevel = "SEV-4"
		summary.IncidentType = "ANOMALY"
	}

	// Key findings
	summary.KeyFindings = append(summary.KeyFindings,
		fmt.Sprintf("Primary Issue: %s (%.1f%% confidence)", diag.PrimaryDetection.Type, diag.PrimaryDetection.Confidence))

	if diag.HealthScore < 70 {
		status := "DEGRADED"
		if diag.HealthScore < 50 {
			status = "CRITICAL"
		}
		summary.KeyFindings = append(summary.KeyFindings,
			fmt.Sprintf("Health Score: %.0f/100 (%s)", diag.HealthScore, status))
	}

	if len(diag.RootCause.AffectedMetrics) > 0 {
		summary.KeyFindings = append(summary.KeyFindings,
			fmt.Sprintf("%d metrics affected", len(diag.RootCause.AffectedMetrics)))
	}

	// Recovery time
	switch diag.PrimaryDetection.Type {
	case DetectionDeploymentBug:
		summary.RecoveryTime = "5-10 minutes (rollback)"
		if diag.RiskLevel == "CRITICAL" {
			summary.EstimatedDowntime = "Active outage"
		}
	case DetectionResourceExhaustion:
		summary.RecoveryTime = "2-5 minutes (scaling)"
	case DetectionMemoryLeak:
		summary.RecoveryTime = "5-15 minutes (restart)"
	case DetectionCascadingFailure:
		summary.RecoveryTime = "15-30 minutes (multi-step)"
	default:
		summary.RecoveryTime = "Minimal"
	}

	// Business impact
	switch diag.RiskLevel {
	case "CRITICAL", "HIGH":
		summary.BusinessImpact = "HIGH - Revenue and SLA impact likely"
	case "MEDIUM":
		summary.BusinessImpact = "MEDIUM - Potential SLA violations"
	default:
		summary.BusinessImpact = "LOW - Minimal business impact"
	}

	return summary
}

// buildDetailedRootCause creates deep root cause analysis
func (ua *UltimateAnalyzer) buildDetailedRootCause(diag *UltimateDiagnosis) *DetailedRootCause {
	rca := &DetailedRootCause{
		PrimaryIssue:        string(diag.PrimaryDetection.Type),
		Confidence:          diag.PrimaryDetection.Confidence,
		PropagationPath:     make([]string, 0),
		ContributingFactors: make([]*ContributingFactor, 0),
		AffectedComponents:  make([]string, 0),
		EvidenceChain:       make([]*Evidence, 0),
	}

	// Time to impact
	rca.TimeToImpact = diag.RootCause.TimeToImpact

	// Trigger event
	rca.TriggerEvent = ua.identifyTrigger(diag)

	// Evidence chain
	rca.EvidenceChain = ua.buildEvidenceChain(diag)

	// Propagation path
	rca.PropagationPath = ua.buildPropagationPath(diag)

	// Blast radius
	rca.BlastRadius = ua.calculateBlastRadius(diag)

	// Contributing factors
	for _, d := range diag.AllDetections {
		if d.Detected && d.Type != diag.PrimaryDetection.Type {
			factor := &ContributingFactor{
				Type:         string(d.Type),
				Description:  d.Recommendation,
				Confidence:   d.Confidence,
				Relationship: ua.determineRelationship(diag.PrimaryDetection.Type, d.Type),
			}
			rca.ContributingFactors = append(rca.ContributingFactors, factor)
		}
	}

	// Affected components
	rca.AffectedComponents = append(rca.AffectedComponents, diag.RootCause.AffectedMetrics...)

	return rca
}

// identifyTrigger identifies what triggered the issue
func (ua *UltimateAnalyzer) identifyTrigger(diag *UltimateDiagnosis) *TriggerEvent {
	features := diag.Features
	trigger := &TriggerEvent{
		Timestamp: diag.Timestamp,
		Details:   make(map[string]interface{}),
	}

	switch diag.PrimaryDetection.Type {
	case DetectionDeploymentBug:
		trigger.Type = "DEPLOYMENT"
		trigger.Description = "Recent deployment introduced bugs causing error spike"
		trigger.Source = "deployment_event"
		trigger.Confidence = diag.PrimaryDetection.Confidence
		trigger.Timestamp = diag.Timestamp.Add(-15 * time.Minute)
		trigger.Details["error_spike"] = features.ErrorRateSpikiness
	case DetectionMemoryLeak:
		trigger.Type = "CODE_CHANGE"
		trigger.Description = "Code change introduced memory leak pattern"
		trigger.Confidence = diag.PrimaryDetection.Confidence * 0.7
		trigger.Timestamp = diag.Timestamp.Add(-1 * time.Hour)
	case DetectionResourceExhaustion:
		if features.CPUVolatility > 30 {
			trigger.Type = "TRAFFIC_SPIKE"
			trigger.Description = "Sudden traffic increase causing resource saturation"
		} else {
			trigger.Type = "GRADUAL_LOAD"
			trigger.Description = "Gradual load increase over time"
		}
		trigger.Confidence = 70.0
	case DetectionExternalFailure:
		trigger.Type = "EXTERNAL_EVENT"
		trigger.Description = "External dependency failure or degradation"
		trigger.Source = "external_service"
		trigger.Confidence = diag.PrimaryDetection.Confidence
	}

	return trigger
}

// buildEvidenceChain builds the evidence chain
func (ua *UltimateAnalyzer) buildEvidenceChain(diag *UltimateDiagnosis) []*Evidence {
	evidence := make([]*Evidence, 0)
	features := diag.Features

	// CPU evidence
	if features.CPUMean > 80 {
		evidence = append(evidence, &Evidence{
			Type:        "THRESHOLD_BREACH",
			Description: fmt.Sprintf("CPU usage exceeded 80%% threshold at %.1f%%", features.CPUMean),
			Metric:      "cpu",
			Value:       features.CPUMean,
			Threshold:   80.0,
			Severity:    SeverityHigh,
			Timestamp:   diag.Timestamp,
			Details:     map[string]interface{}{"trend": features.CPUTrend, "volatility": features.CPUVolatility},
		})
	}

	// Memory evidence
	if features.MemoryMean > 80 {
		evidence = append(evidence, &Evidence{
			Type:        "THRESHOLD_BREACH",
			Description: fmt.Sprintf("Memory usage exceeded 80%% threshold at %.1f%%", features.MemoryMean),
			Metric:      "memory",
			Value:       features.MemoryMean,
			Threshold:   80.0,
			Severity:    SeverityHigh,
			Timestamp:   diag.Timestamp,
			Details:     map[string]interface{}{"trend": features.MemoryTrend},
		})
	}

	// Error rate evidence
	if features.ErrorRateMean > 10 {
		evidence = append(evidence, &Evidence{
			Type:        "METRIC_ANOMALY",
			Description: fmt.Sprintf("Error rate anomaly at %.1f/min", features.ErrorRateMean),
			Metric:      "error_rate",
			Value:       features.ErrorRateMean,
			Threshold:   10.0,
			Severity:    SeverityCritical,
			Timestamp:   diag.Timestamp,
			Details:     map[string]interface{}{"spikiness": features.ErrorRateSpikiness},
		})
	}

	// Correlation evidence
	if math.Abs(features.CPUErrorCorr) > 0.7 {
		evidence = append(evidence, &Evidence{
			Type:        "CORRELATION",
			Description: fmt.Sprintf("Strong CPU-Error correlation (%.2f) indicates resource saturation", features.CPUErrorCorr),
			Severity:    SeverityHigh,
			Timestamp:   diag.Timestamp,
			Details:     map[string]interface{}{"correlation": features.CPUErrorCorr},
		})
	}

	return evidence
}

// buildPropagationPath shows how the issue spread
func (ua *UltimateAnalyzer) buildPropagationPath(diag *UltimateDiagnosis) []string {
	path := make([]string, 0)
	features := diag.Features

	switch diag.PrimaryDetection.Type {
	case DetectionDeploymentBug:
		path = append(path, "1. Deployment initiated")
		path = append(path, "2. New code introduced bugs")
		path = append(path, "3. Error rate spiked dramatically")
		if features.CPUMean > 70 {
			path = append(path, "4. Error handling consumed CPU resources")
		}
	case DetectionMemoryLeak:
		path = append(path, "1. Memory leak introduced")
		path = append(path, "2. Memory usage gradually increased")
		path = append(path, "3. GC pressure increased")
	case DetectionResourceExhaustion:
		path = append(path, "1. Resource demand increased")
		path = append(path, "2. CPU/Memory approaching limits")
		path = append(path, "3. Request queueing and timeouts")
	}

	return path
}

// calculateBlastRadius calculates impact scope
func (ua *UltimateAnalyzer) calculateBlastRadius(diag *UltimateDiagnosis) *BlastRadius {
	radius := &BlastRadius{
		AffectedServices: []string{diag.ServiceName},
		DownstreamImpact: make([]string, 0),
		UpstreamImpact:   make([]string, 0),
	}

	// Determine scope
	if diag.HealthScore < 30 {
		radius.Scope = "CLUSTER"
		radius.EstimatedReach = 1000
	} else if diag.HealthScore < 50 {
		radius.Scope = "NAMESPACE"
		radius.EstimatedReach = 500
	} else {
		radius.Scope = "SERVICE"
		radius.EstimatedReach = 100
	}

	// Affected users
	features := diag.Features
	if features.ErrorRateMean > 50 {
		radius.AffectedUsers = "> 50% of users experiencing errors"
	} else if features.ErrorRateMean > 20 {
		radius.AffectedUsers = "20-50% of users affected"
	} else if features.ErrorRateMean > 5 {
		radius.AffectedUsers = "5-20% experiencing issues"
	} else {
		radius.AffectedUsers = "< 5% minimal impact"
	}

	// Downstream impact
	if diag.RiskLevel == "CRITICAL" || diag.RiskLevel == "HIGH" {
		radius.DownstreamImpact = append(radius.DownstreamImpact,
			"Dependent services may experience degradation",
			"API consumers may see increased latency/errors")
	}

	return radius
}

// determineRelationship determines how issues relate
func (ua *UltimateAnalyzer) determineRelationship(primary, secondary DetectionType) string {
	relationships := map[string]map[DetectionType]string{
		string(DetectionDeploymentBug): {
			DetectionResourceExhaustion: "causing resource spike",
			DetectionCascadingFailure:   "triggering cascade",
		},
		string(DetectionResourceExhaustion): {
			DetectionMemoryLeak: "likely caused by memory leak",
		},
	}

	if rels, ok := relationships[string(primary)]; ok {
		if rel, ok := rels[secondary]; ok {
			return rel
		}
	}
	return "may be related"
}

// buildTimeline creates diagnostic timeline
func (ua *UltimateAnalyzer) buildTimeline(diag *UltimateDiagnosis) *DiagnosticTimeline {
	timeline := &DiagnosticTimeline{
		StartTime:     diag.Timestamp.Add(-30 * time.Minute),
		DetectionTime: diag.Timestamp,
		Events:        make([]*TimelineEvent, 0),
		KeyMilestones: make([]string, 0),
	}

	// Add key events
	features := diag.Features

	if diag.PrimaryDetection.Type == DetectionDeploymentBug {
		timeline.Events = append(timeline.Events, &TimelineEvent{
			Timestamp:   diag.Timestamp.Add(-15 * time.Minute),
			Type:        "DEPLOYMENT",
			Description: "New deployment detected",
			Severity:    SeverityHigh,
		})
		timeline.KeyMilestones = append(timeline.KeyMilestones, "Deployment 15 minutes ago")
	}

	if features.ErrorRateMean > 10 {
		timeline.Events = append(timeline.Events, &TimelineEvent{
			Timestamp:   diag.Timestamp.Add(-5 * time.Minute),
			Type:        "METRIC_CHANGE",
			Description: fmt.Sprintf("Error rate spike to %.1f/min", features.ErrorRateMean),
			Severity:    SeverityCritical,
		})
	}

	// Prediction window
	timeline.PredictionWindow = ua.buildPredictionWindow(diag)

	return timeline
}

// buildPredictionWindow creates predictions
func (ua *UltimateAnalyzer) buildPredictionWindow(diag *UltimateDiagnosis) *PredictionWindow {
	window := &PredictionWindow{
		ConfidenceLevel: diag.PredictabilityScore,
	}

	features := diag.Features

	// Memory prediction
	if features.MemoryTrend > 0.1 {
		predictedIn1h := features.MemoryMean + (features.MemoryTrend * 60)
		window.Next1Hour = &Prediction{
			Metric:             "memory",
			CurrentValue:       features.MemoryMean,
			PredictedValue:     math.Min(predictedIn1h, 100),
			ConfidenceInterval: [2]float64{predictedIn1h * 0.9, math.Min(predictedIn1h*1.1, 100)},
			Trend:              "INCREASING",
			Likelihood:         math.Min(diag.PredictabilityScore, 90),
		}

		if predictedIn1h > 90 {
			window.Next1Hour.RecommendedAction = "Scale or increase memory limits before exhaustion"
		}
	}

	return window
}

// buildEnhancedActions creates enhanced actuator actions
func (ua *UltimateAnalyzer) buildEnhancedActions(diag *UltimateDiagnosis) []*EnhancedActuatorAction {
	enhanced := make([]*EnhancedActuatorAction, 0)

	// Convert basic actions to enhanced
	for _, action := range diag.ActuatorActions {
		enhancedAction := &EnhancedActuatorAction{
			ActionType:   action.ActionType,
			Priority:     action.Priority,
			TargetMetric: action.TargetMetric,
			CurrentValue: action.CurrentValue,
			TargetValue:  action.TargetValue,
			Reason:       action.Reason,
			Confidence:   action.Confidence,
			Parameters:   action.Parameters,
		}

		// Add safety features for ROLLBACK
		if action.ActionType == "ROLLBACK" {
			enhancedAction.PreConditions = []string{
				"Previous stable version available",
				"No breaking schema changes",
				"Deployment within last 24 hours",
			}
			enhancedAction.PostConditions = []string{
				"Error rate < 5/min",
				"All pods healthy",
				"Latency within SLA",
			}
			enhancedAction.SuccessCriteria = []*SuccessCriterion{
				{Metric: "error_rate", Operator: "<=", Threshold: 5.0, Duration: "5m", Priority: "REQUIRED"},
				{Metric: "pod_ready_ratio", Operator: ">=", Threshold: 1.0, Duration: "2m", Priority: "REQUIRED"},
			}
			enhancedAction.RollbackPlan = &RollbackPlan{
				CanRollback:      true,
				RollbackAction:   "FORWARD_FIX",
				AutoRollback:     true,
				RollbackTriggers: []string{"Errors persist after rollback"},
			}
			enhancedAction.EstimatedImpact = &ActionImpact{
				UserImpact:         "BRIEF",
				AvailabilityImpact: "30-60s disruption during rollback",
				PerformanceImpact:  "Expected improvement",
				Duration:           "2-3 minutes",
				Reversible:         true,
			}
			enhancedAction.TimeWindow = &TimeWindow{
				Earliest:  diag.Timestamp,
				Latest:    diag.Timestamp.Add(5 * time.Minute),
				Preferred: diag.Timestamp.Add(1 * time.Minute),
				Urgency:   "NOW",
				CanDelay:  false,
			}
		}

		enhanced = append(enhanced, enhancedAction)
	}

	return enhanced
}

// buildHealthIntelligence creates health intelligence
func (ua *UltimateAnalyzer) buildHealthIntelligence(diag *UltimateDiagnosis) *HealthIntelligence {
	// Calculate health history
	history := &HealthHistory{
		Last5Minutes:  diag.HealthScore + 5, // Estimate
		Last15Minutes: diag.HealthScore + 10,
		Last30Minutes: diag.HealthScore + 15,
		Last1Hour:     diag.HealthScore + 20,
	}

	// Determine trend
	trend := "STABLE"
	degradationRate := 0.0
	if diag.HealthScore < 70 {
		trend = "DEGRADING"
		degradationRate = -0.5 // points per minute
	}

	return &HealthIntelligence{
		CurrentHealth:   diag.HealthScore,
		HealthHistory:   history,
		HealthTrend:     trend,
		SystemStress:    diag.SystemStress,
		StabilityIndex:  diag.StabilityIndex,
		Predictability:  diag.PredictabilityScore,
		AnomalyScore:    ua.calculateAnomalyScore(diag),
		DegradationRate: degradationRate,
	}
}

// calculateAnomalyScore calculates anomaly score
func (ua *UltimateAnalyzer) calculateAnomalyScore(diag *UltimateDiagnosis) float64 {
	score := 0.0
	features := diag.Features

	if diag.PrimaryDetection.Detected {
		score += diag.PrimaryDetection.Confidence * 0.4
	}

	if features.CPUMean > 80 {
		score += 20
	}
	if features.MemoryMean > 80 {
		score += 20
	}
	if features.ErrorRateMean > 50 {
		score += 30
	}

	return math.Min(score, 100)
}

// buildSLACompliance creates SLA compliance data
func (ua *UltimateAnalyzer) buildSLACompliance(diag *UltimateDiagnosis) *SLACompliance {
	features := diag.Features
	compliance := &SLACompliance{
		Metrics: make(map[string]*SLAMetric),
	}

	// Availability SLA
	availPct := 100.0 - (features.ErrorRateMean / 10.0)
	availStatus := "GOOD"
	if availPct < 99.0 {
		availStatus = "CRITICAL"
	} else if availPct < 99.5 {
		availStatus = "WARNING"
	}

	compliance.Metrics["availability"] = &SLAMetric{
		Name:    "availability",
		Target:  99.9,
		Current: availPct,
		Status:  availStatus,
		Margin:  availPct - 99.9,
		Trend:   "STABLE",
	}

	// Error rate SLA
	errorStatus := "GOOD"
	if features.ErrorRateMean > 50 {
		errorStatus = "CRITICAL"
		compliance.ViolationCount++
	} else if features.ErrorRateMean > 10 {
		errorStatus = "WARNING"
		compliance.WarningCount++
	}

	compliance.Metrics["error_rate"] = &SLAMetric{
		Name:    "error_rate",
		Target:  1.0,
		Current: features.ErrorRateMean,
		Status:  errorStatus,
		Margin:  1.0 - features.ErrorRateMean,
	}

	// Overall status
	if compliance.ViolationCount > 0 {
		compliance.OverallStatus = "BREACHED"
		compliance.TimeToBreach = "ALREADY BREACHED"
		compliance.BreachProbability = 100
	} else if compliance.WarningCount > 0 {
		compliance.OverallStatus = "AT_RISK"
		compliance.TimeToBreach = "< 1 hour"
		compliance.BreachProbability = 60
	} else {
		compliance.OverallStatus = "COMPLIANT"
		compliance.BreachProbability = 10
	}

	return compliance
}

// buildMetricIntelligence creates metric intelligence
func (ua *UltimateAnalyzer) buildMetricIntelligence(diag *UltimateDiagnosis) *MetricIntelligence {
	features := diag.Features
	intel := &MetricIntelligence{
		Correlations:      make([]*MetricCorrelation, 0),
		AnomalousMetrics:  make([]string, 0),
		TrendingMetrics:   make([]*TrendingMetric, 0),
		ThresholdBreaches: make([]*ThresholdBreach, 0),
	}

	// CPU-Error correlation
	if math.Abs(features.CPUErrorCorr) > 0.7 {
		intel.Correlations = append(intel.Correlations, &MetricCorrelation{
			Metric1:      "cpu",
			Metric2:      "errors",
			Correlation:  features.CPUErrorCorr,
			Strength:     "STRONG",
			Causality:    "LIKELY_CAUSE",
			Explanation:  "High CPU causing errors via saturation",
			Significance: math.Abs(features.CPUErrorCorr) * 100,
		})
	}

	// Anomalous metrics
	if features.ErrorRateMean > 10 {
		intel.AnomalousMetrics = append(intel.AnomalousMetrics, "error_rate")
	}
	if features.CPUMean > 80 {
		intel.AnomalousMetrics = append(intel.AnomalousMetrics, "cpu")
	}

	// Trending metrics
	if features.MemoryTrend > 0.5 {
		intel.TrendingMetrics = append(intel.TrendingMetrics, &TrendingMetric{
			Metric:    "memory",
			Current:   features.MemoryMean,
			Trend:     features.MemoryTrend,
			Direction: "INCREASING",
			Concern:   "HIGH",
		})
	}

	// Threshold breaches
	if features.CPUMean > 80 {
		intel.ThresholdBreaches = append(intel.ThresholdBreaches, &ThresholdBreach{
			Metric:    "cpu",
			Threshold: 80.0,
			Current:   features.CPUMean,
			Severity:  SeverityHigh,
			Timestamp: diag.Timestamp,
		})
	}

	return intel
}

// buildImpactAnalysis creates impact analysis
func (ua *UltimateAnalyzer) buildImpactAnalysis(diag *UltimateDiagnosis) *ImpactAnalysis {
	features := diag.Features

	impact := &ImpactAnalysis{}

	// User impact
	if features.ErrorRateMean > 50 {
		impact.UserImpact = "SEVERE"
		impact.AffectedUsersPct = "> 50%"
	} else if features.ErrorRateMean > 20 {
		impact.UserImpact = "HIGH"
		impact.AffectedUsersPct = "20-50%"
	} else if features.ErrorRateMean > 5 {
		impact.UserImpact = "MODERATE"
		impact.AffectedUsersPct = "5-20%"
	} else {
		impact.UserImpact = "LOW"
		impact.AffectedUsersPct = "< 5%"
	}

	// Performance score
	impact.PerformanceScore = diag.HealthScore

	// Business impact
	switch diag.RiskLevel {
	case "CRITICAL":
		impact.BusinessImpact = "HIGH"
		impact.RevenueImpact = "Active revenue loss"
	case "HIGH":
		impact.BusinessImpact = "MEDIUM"
		impact.RevenueImpact = "Potential revenue impact"
	default:
		impact.BusinessImpact = "LOW"
	}

	// Cost impact
	if diag.SystemStress > 90 {
		impact.CostImpact = "HIGH - Scaling required"
	} else {
		impact.CostImpact = "NORMAL"
	}

	// Recovery difficulty
	switch diag.PrimaryDetection.Type {
	case DetectionDeploymentBug:
		impact.RecoveryDifficulty = "EASY - Rollback available"
	case DetectionMemoryLeak:
		impact.RecoveryDifficulty = "HARD - Requires code fix"
	default:
		impact.RecoveryDifficulty = "MEDIUM"
	}

	// Availability impact
	if diag.HealthScore < 30 {
		impact.AvailabilityImpact = "CRITICAL - Service failing"
		impact.EstimatedDowntime = "Active outage"
	} else if diag.HealthScore < 50 {
		impact.AvailabilityImpact = "HIGH - Severe degradation"
	} else if diag.HealthScore < 70 {
		impact.AvailabilityImpact = "MEDIUM - Performance degraded"
	} else {
		impact.AvailabilityImpact = "LOW - Minor impact"
	}

	return impact
}
