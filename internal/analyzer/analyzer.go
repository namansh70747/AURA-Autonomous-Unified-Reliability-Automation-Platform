package analyzer

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/namansh70747/AURA-Autonomous-Unified-Reliability-Automation-Platform/internal/storage"
	"github.com/namansh70747/AURA-Autonomous-Unified-Reliability-Automation-Platform/pkg/logger"
	"go.uber.org/zap"
)

type Analyzer struct {
	db                         *storage.PostgresClient
	memoryLeakDetector         *MemoryLeakDetector
	deploymentBugDetector      *DeploymentBugDetector
	cascadeDetector            *CascadeDetector
	externalFailureDetector    *ExternalFailureDetector
	resourceExhaustionDetector *ResourceExhaustionDetector
	patternMatcher             *PatternMatcher
	anomalyDetector            *AnomalyDetector
	serviceCorrelator          *ServiceCorrelator
}

func NewAnalyzer(db *storage.PostgresClient) *Analyzer {
	logger.Info("Initializing pattern analyzer with advanced features")

	return &Analyzer{
		db:                         db,
		memoryLeakDetector:         NewMemoryLeakDetector(db),
		deploymentBugDetector:      NewDeploymentBugDetector(db),
		cascadeDetector:            NewCascadeDetector(db),
		externalFailureDetector:    NewExternalFailureDetector(db),
		resourceExhaustionDetector: NewResourceExhaustionDetector(db),
		patternMatcher:             NewPatternMatcher(db),
		anomalyDetector:            NewAnomalyDetector(db),
		serviceCorrelator:          NewServiceCorrelator(db),
	}
}

func (a *Analyzer) AnalyzeService(ctx context.Context, serviceName string) (*Diagnosis, error) {
	logger.Info("Starting pattern analysis",
		zap.String("service", serviceName),
	)

	results := make(chan *Detection, 5)
	errors := make(chan error, 5)

	go func() {
		detection, err := a.memoryLeakDetector.Analyze(ctx, serviceName)
		if err != nil {
			errors <- err
			return
		}
		results <- detection
	}()

	go func() {
		detection, err := a.deploymentBugDetector.Analyze(ctx, serviceName)
		if err != nil {
			errors <- err
			return
		}
		results <- detection
	}()

	go func() {
		detection, err := a.cascadeDetector.Analyze(ctx, serviceName)
		if err != nil {
			errors <- err
			return
		}
		results <- detection
	}()

	go func() {
		detection, err := a.externalFailureDetector.Analyze(ctx, serviceName)
		if err != nil {
			errors <- err
			return
		}
		results <- detection
	}()

	go func() {
		detection, err := a.resourceExhaustionDetector.Analyze(ctx, serviceName)
		if err != nil {
			errors <- err
			return
		}
		results <- detection
	}()

	detections := []*Detection{}
	for i := 0; i < 5; i++ {
		select {
		case detection := <-results:
			detections = append(detections, detection)
			logger.Debug("Detection completed",
				zap.String("service", serviceName),
				zap.String("type", string(detection.Type)),
				zap.Bool("detected", detection.Detected),
				zap.Float64("confidence", detection.Confidence),
			)
		case err := <-errors:
			logger.Error("Detection failed",
				zap.String("service", serviceName),
				zap.Error(err),
			)
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	sort.Slice(detections, func(i, j int) bool {
		return detections[i].Confidence > detections[j].Confidence
	})

	bestMatch := detections[0]

	diagnosis := &Diagnosis{
		ServiceName:         serviceName,
		Problem:             DetectionHealthy,
		Confidence:          0,
		Timestamp:           time.Now(),
		Evidence:            map[string]interface{}{},
		Recommendation:      "Service is healthy. No issues detected.",
		Severity:            "LOW",
		AllDetections:       make([]Detection, 0),
		MultipleProblems:    false,
		HighConfidenceCount: 0,
	}

	for _, d := range detections {
		diagnosis.AllDetections = append(diagnosis.AllDetections, *d)
	}

	// Find ALL problems with confidence > 80% (not just the best match)
	highConfidenceDetections := []*Detection{}
	for _, d := range detections {
		if d.Detected && d.Confidence > 80.0 {
			highConfidenceDetections = append(highConfidenceDetections, d)
		}
	}

	if len(highConfidenceDetections) > 0 {
		// Use best match for main diagnosis
		diagnosis.Problem = bestMatch.Type
		diagnosis.Confidence = bestMatch.Confidence
		diagnosis.Evidence = bestMatch.Evidence
		diagnosis.Recommendation = bestMatch.Recommendation
		diagnosis.Severity = bestMatch.Severity
		diagnosis.HighConfidenceCount = len(highConfidenceDetections)
		diagnosis.MultipleProblems = len(highConfidenceDetections) > 1

		// If multiple problems, add note to recommendation
		if diagnosis.MultipleProblems {
			problemList := ""
			for i, d := range highConfidenceDetections {
				if i > 0 {
					problemList += ", "
				}
				problemList += string(d.Type)
			}
			diagnosis.Recommendation = diagnosis.Recommendation +
				" | ALERT: Multiple issues detected (" + problemList + "). Check all_detections for details."
		}

		// Log all high-confidence problems
		if len(highConfidenceDetections) > 1 {
			logger.Warn("Multiple problems detected",
				zap.String("service", serviceName),
				zap.Int("problem_count", len(highConfidenceDetections)),
			)
		}

		// Save ALL high-confidence detections to database
		if a.db != nil {
			savedCount := 0
			for _, detection := range highConfidenceDetections {
				diagnosisRecord := &storage.DiagnosisRecord{
					ServiceName:    serviceName,
					ProblemType:    string(detection.Type),
					Confidence:     detection.Confidence,
					Severity:       detection.Severity,
					Evidence:       detection.Evidence,
					Recommendation: detection.Recommendation,
					Timestamp:      time.Now(),
				}

				if err := a.db.SaveDiagnosis(ctx, diagnosisRecord); err != nil {
					logger.Error("Failed to save diagnosis",
						zap.String("problem", string(detection.Type)),
						zap.Error(err),
					)
				} else {
					savedCount++
				}
			}

			logger.Info("Diagnoses saved",
				zap.String("service", serviceName),
				zap.Int("saved_count", savedCount),
				zap.Int("total_detected", len(highConfidenceDetections)),
			)
		}

		// Log primary problem
		logger.Info("Problem detected",
			zap.String("service", serviceName),
			zap.String("primary_problem", string(bestMatch.Type)),
			zap.Float64("confidence", bestMatch.Confidence),
			zap.String("severity", bestMatch.Severity),
			zap.Int("total_problems", len(highConfidenceDetections)),
		)
	} else {
		logger.Info("Service healthy",
			zap.String("service", serviceName),
			zap.Float64("highest_confidence", bestMatch.Confidence),
		)
	}

	return diagnosis, nil
}

func (a *Analyzer) AnalyzeAllServices(ctx context.Context, services []string) (map[string]*Diagnosis, error) {
	logger.Info("Analyzing all services",
		zap.Int("count", len(services)),
	)

	results := make(map[string]*Diagnosis)

	for _, service := range services {
		diagnosis, err := a.AnalyzeService(ctx, service)
		if err != nil {
			logger.Error("Failed to analyze service",
				zap.String("service", service),
				zap.Error(err),
			)
			continue
		}
		results[service] = diagnosis
	}

	return results, nil
}

// ==================== ADVANCED ANALYSIS METHODS ====================

// AnalyzeServiceAdvanced performs deep analysis with cross-detector correlation
func (a *Analyzer) AnalyzeServiceAdvanced(ctx context.Context, serviceName string) (*AdvancedDiagnosis, error) {
	logger.Info("Starting advanced pattern analysis",
		zap.String("service", serviceName),
	)

	// Run standard analysis first
	basicDiagnosis, err := a.AnalyzeService(ctx, serviceName)
	if err != nil {
		return nil, err
	}

	// Perform advanced analytics in parallel
	var wg sync.WaitGroup
	var mu sync.Mutex

	advDiag := &AdvancedDiagnosis{
		BasicDiagnosis: basicDiagnosis,
		RootCause:      "",
		ImpactScore:    0,
		TrendAnalysis:  map[string]string{},
		Correlations:   []CorrelationInsight{},
		PriorityScore:  0,
	}

	// 1. Analyze root cause based on detection patterns
	wg.Add(1)
	go func() {
		defer wg.Done()
		rootCause := a.analyzeRootCause(basicDiagnosis)
		mu.Lock()
		advDiag.RootCause = rootCause
		mu.Unlock()
	}()

	// 2. Calculate impact score
	wg.Add(1)
	go func() {
		defer wg.Done()
		impact := a.calculateImpactScore(basicDiagnosis)
		mu.Lock()
		advDiag.ImpactScore = impact
		mu.Unlock()
	}()

	// 3. Analyze trends for key metrics
	wg.Add(1)
	go func() {
		defer wg.Done()
		trends := a.analyzeTrends(ctx, serviceName)
		mu.Lock()
		advDiag.TrendAnalysis = trends
		mu.Unlock()
	}()

	// 4. Find cross-detector correlations
	wg.Add(1)
	go func() {
		defer wg.Done()
		correlations := a.findCrossDetectorCorrelations(basicDiagnosis)
		mu.Lock()
		advDiag.Correlations = correlations
		mu.Unlock()
	}()

	// 5. Calculate priority score for triage
	wg.Add(1)
	go func() {
		defer wg.Done()
		priority := a.calculatePriorityScore(basicDiagnosis)
		mu.Lock()
		advDiag.PriorityScore = priority
		mu.Unlock()
	}()

	wg.Wait()

	logger.Info("Advanced analysis complete",
		zap.String("service", serviceName),
		zap.String("root_cause", advDiag.RootCause),
		zap.Float64("impact_score", advDiag.ImpactScore),
		zap.Float64("priority_score", advDiag.PriorityScore),
	)

	return advDiag, nil
}

// analyzeRootCause determines the most likely root cause from detection patterns
func (a *Analyzer) analyzeRootCause(diag *Diagnosis) string {
	if diag.Problem == DetectionHealthy {
		return "No issues detected"
	}

	// Pattern-based root cause analysis
	detectionMap := make(map[DetectionType]*Detection)
	for i := range diag.AllDetections {
		d := &diag.AllDetections[i]
		detectionMap[d.Type] = d
	}

	// Rule 1: Memory leak + Resource exhaustion = Memory management issue
	if memLeak, ok := detectionMap[DetectionMemoryLeak]; ok && memLeak.Detected && memLeak.Confidence > 70 {
		if resExh, ok := detectionMap[DetectionResourceExhaustion]; ok && resExh.Detected {
			return "Memory management issue - Suspected memory leak causing resource exhaustion"
		}
		return "Memory leak - Application not releasing memory properly"
	}

	// Rule 2: Deployment bug + Cascade = Bad deployment causing ripple effects
	if depBug, ok := detectionMap[DetectionDeploymentBug]; ok && depBug.Detected && depBug.Confidence > 70 {
		if cascade, ok := detectionMap[DetectionCascadingFailure]; ok && cascade.Detected {
			return "Bad deployment with cascading impact - Rollback recommended"
		}
		return "Recent deployment introduced bugs - Code quality issue"
	}

	// Rule 3: External failure + Cascade = Upstream dependency failure
	if extFail, ok := detectionMap[DetectionExternalFailure]; ok && extFail.Detected && extFail.Confidence > 70 {
		if cascade, ok := detectionMap[DetectionCascadingFailure]; ok && cascade.Detected {
			return "Upstream dependency failure cascading to dependent services"
		}
		return "External service dependency issue - Third-party service degradation"
	}

	// Rule 4: Resource exhaustion + High traffic = Scaling issue
	if resExh, ok := detectionMap[DetectionResourceExhaustion]; ok && resExh.Detected {
		if evidence, ok := resExh.Evidence["traffic_high"].(bool); ok && evidence {
			return "Capacity issue - Service needs scaling to handle traffic load"
		}
		return "Resource leak or inefficient resource usage"
	}

	// Rule 5: Cascade alone = Upstream service issue
	if cascade, ok := detectionMap[DetectionCascadingFailure]; ok && cascade.Detected && cascade.Confidence > 70 {
		return "Cascading failure - Check upstream dependencies"
	}

	// Default to primary detection
	return fmt.Sprintf("%s detected - %s", diag.Problem, diag.Recommendation)
}

// calculateImpactScore quantifies the severity and scope of detected issues
func (a *Analyzer) calculateImpactScore(diag *Diagnosis) float64 {
	if diag.Problem == DetectionHealthy {
		return 0.0
	}

	score := 0.0

	// Base score from confidence and severity
	score += diag.Confidence * 0.4 // 40% weight to confidence

	severityWeight := map[string]float64{
		"LOW":      10.0,
		"MEDIUM":   30.0,
		"HIGH":     60.0,
		"CRITICAL": 100.0,
	}
	score += severityWeight[diag.Severity] * 0.3 // 30% weight to severity

	// Multiple problems increase impact
	if diag.MultipleProblems {
		score += float64(diag.HighConfidenceCount) * 10.0 // +10 per additional problem
	}

	// Problem type modifiers
	typeImpact := map[DetectionType]float64{
		DetectionCascadingFailure:   1.5, // Cascades affect multiple services
		DetectionResourceExhaustion: 1.3, // Can lead to outages
		DetectionMemoryLeak:         1.2, // Degrades over time
		DetectionDeploymentBug:      1.1, // Recent changes are risky
		DetectionExternalFailure:    1.0, // External dependencies
	}

	if multiplier, ok := typeImpact[diag.Problem]; ok {
		score *= multiplier
	}

	// Cap at 100
	if score > 100 {
		score = 100
	}

	return math.Round(score*100) / 100
}

// analyzeTrends examines metric trends for key indicators
func (a *Analyzer) analyzeTrends(ctx context.Context, serviceName string) map[string]string {
	trends := make(map[string]string)

	// Check context cancellation
	select {
	case <-ctx.Done():
		logger.Warn("Trend analysis cancelled", zap.String("service", serviceName))
		return trends
	default:
	}

	keyMetrics := []string{"cpu_usage", "memory_usage", "error_rate", "response_time", "request_rate"}

	for _, metric := range keyMetrics {
		result, err := a.patternMatcher.DetectTrend(serviceName, metric, 30*time.Minute)
		if err != nil {
			trends[metric] = "unknown"
			continue
		}

		if result != nil {
			trends[metric] = result.Direction
		} else {
			trends[metric] = "stable"
		}
	}

	return trends
} // findCrossDetectorCorrelations identifies relationships between different detections
func (a *Analyzer) findCrossDetectorCorrelations(diag *Diagnosis) []CorrelationInsight {
	correlations := []CorrelationInsight{}

	detectedProblems := []*Detection{}
	for i := range diag.AllDetections {
		if diag.AllDetections[i].Detected && diag.AllDetections[i].Confidence > 60 {
			detectedProblems = append(detectedProblems, &diag.AllDetections[i])
		}
	}

	if len(detectedProblems) < 2 {
		return correlations
	}

	// Analyze pairwise correlations
	for i := 0; i < len(detectedProblems)-1; i++ {
		for j := i + 1; j < len(detectedProblems); j++ {
			p1 := detectedProblems[i]
			p2 := detectedProblems[j]

			correlation := a.analyzeDetectionCorrelation(p1, p2)
			if correlation != nil {
				correlations = append(correlations, *correlation)
			}
		}
	}

	return correlations
}

// analyzeDetectionCorrelation finds relationships between two detections
func (a *Analyzer) analyzeDetectionCorrelation(d1, d2 *Detection) *CorrelationInsight {
	// Known correlation patterns
	patterns := map[string]map[string]CorrelationInsight{
		string(DetectionMemoryLeak): {
			string(DetectionResourceExhaustion): {
				Detector1:   string(DetectionMemoryLeak),
				Detector2:   string(DetectionResourceExhaustion),
				Correlation: 0.85,
				Explanation: "Memory leak directly causes resource exhaustion",
				Causality:   "Memory Leak → Resource Exhaustion",
			},
		},
		string(DetectionDeploymentBug): {
			string(DetectionCascadingFailure): {
				Detector1:   string(DetectionDeploymentBug),
				Detector2:   string(DetectionCascadingFailure),
				Correlation: 0.75,
				Explanation: "Buggy deployment causing cascade to downstream services",
				Causality:   "Deployment Bug → Cascade Failure",
			},
		},
		string(DetectionExternalFailure): {
			string(DetectionCascadingFailure): {
				Detector1:   string(DetectionExternalFailure),
				Detector2:   string(DetectionCascadingFailure),
				Correlation: 0.80,
				Explanation: "External dependency failure propagating through services",
				Causality:   "External Failure → Cascade",
			},
		},
		string(DetectionResourceExhaustion): {
			string(DetectionCascadingFailure): {
				Detector1:   string(DetectionResourceExhaustion),
				Detector2:   string(DetectionCascadingFailure),
				Correlation: 0.70,
				Explanation: "Resource exhaustion causing downstream cascade",
				Causality:   "Resource Exhaustion → Cascade",
			},
		},
	}

	// Check both directions
	if correlations, ok := patterns[string(d1.Type)]; ok {
		if insight, ok := correlations[string(d2.Type)]; ok {
			return &insight
		}
	}

	if correlations, ok := patterns[string(d2.Type)]; ok {
		if insight, ok := correlations[string(d1.Type)]; ok {
			return &insight
		}
	}

	return nil
}

// calculatePriorityScore determines urgency for incident response
func (a *Analyzer) calculatePriorityScore(diag *Diagnosis) float64 {
	if diag.Problem == DetectionHealthy {
		return 0.0
	}

	score := 0.0

	// Severity contributes heavily to priority
	severityScore := map[string]float64{
		"CRITICAL": 40.0,
		"HIGH":     30.0,
		"MEDIUM":   15.0,
		"LOW":      5.0,
	}
	score += severityScore[diag.Severity]

	// Confidence in detection
	score += (diag.Confidence / 100) * 20.0

	// Multiple problems increase priority
	if diag.MultipleProblems {
		score += float64(diag.HighConfidenceCount-1) * 5.0
	}

	// Certain problem types are higher priority
	urgencyModifier := map[DetectionType]float64{
		DetectionCascadingFailure:   1.4, // Affects multiple services
		DetectionResourceExhaustion: 1.3, // Imminent outage risk
		DetectionDeploymentBug:      1.2, // Recent changes need quick action
		DetectionMemoryLeak:         1.1, // Gradual degradation
		DetectionExternalFailure:    1.0, // May be out of control
	}

	if modifier, ok := urgencyModifier[diag.Problem]; ok {
		score *= modifier
	}

	// Cap at 100
	if score > 100 {
		score = 100
	}

	return math.Round(score*100) / 100
}

// GetHealthScore returns an overall health score (0-100) for a service
func (a *Analyzer) GetHealthScore(ctx context.Context, serviceName string) (float64, error) {
	diagnosis, err := a.AnalyzeService(ctx, serviceName)
	if err != nil {
		return 0, err
	}

	if diagnosis.Problem == DetectionHealthy {
		return 100.0, nil
	}

	// Start at 100 and deduct based on issues
	healthScore := 100.0

	// Primary problem impact
	severityDeduction := map[string]float64{
		"CRITICAL": 50.0,
		"HIGH":     30.0,
		"MEDIUM":   15.0,
		"LOW":      5.0,
	}

	if deduction, ok := severityDeduction[diagnosis.Severity]; ok {
		healthScore -= deduction * (diagnosis.Confidence / 100)
	}

	// Additional problems compound the impact
	if diagnosis.MultipleProblems {
		healthScore -= float64(diagnosis.HighConfidenceCount-1) * 10.0
	}

	// Floor at 0
	if healthScore < 0 {
		healthScore = 0
	}

	return math.Round(healthScore*100) / 100, nil
}

// CompareServices compares health across multiple services
func (a *Analyzer) CompareServices(ctx context.Context, services []string) ([]ServiceComparison, error) {
	comparisons := []ServiceComparison{}

	for _, service := range services {
		health, err := a.GetHealthScore(ctx, service)
		if err != nil {
			logger.Error("Failed to get health score",
				zap.String("service", service),
				zap.Error(err),
			)
			continue
		}

		diagnosis, _ := a.AnalyzeService(ctx, service)

		comparison := ServiceComparison{
			ServiceName:       service,
			HealthScore:       health,
			PrimaryIssue:      string(diagnosis.Problem),
			IssueCount:        diagnosis.HighConfidenceCount,
			Severity:          diagnosis.Severity,
			RequiresAttention: health < 80.0,
		}

		comparisons = append(comparisons, comparison)
	}

	// Sort by health score (worst first)
	sort.Slice(comparisons, func(i, j int) bool {
		return comparisons[i].HealthScore < comparisons[j].HealthScore
	})

	return comparisons, nil
}
