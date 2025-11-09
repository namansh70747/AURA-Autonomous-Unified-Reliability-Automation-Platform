package analyzer

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/namansh70747/AURA-Autonomous-Unified-Reliability-Automation-Platform/internal/storage"
	"github.com/namansh70747/AURA-Autonomous-Unified-Reliability-Automation-Platform/pkg/logger"
	"go.uber.org/zap"
)

type DeploymentBugDetector struct {
	db              *storage.PostgresClient
	patternMatcher  *PatternMatcher
	anomalyDetector *AnomalyDetector
}

func NewDeploymentBugDetector(db *storage.PostgresClient) *DeploymentBugDetector {
	return &DeploymentBugDetector{
		db:              db,
		patternMatcher:  NewPatternMatcher(db),
		anomalyDetector: NewAnomalyDetector(db),
	}
}

func (d *DeploymentBugDetector) Analyze(ctx context.Context, serviceName string) (*Detection, error) {
	deploymentTime := time.Now().Add(-15 * time.Minute)
	return d.AnalyzeWithDeploymentTime(ctx, serviceName, deploymentTime)
}

// AnalyzeWithDeploymentTime detects deployment-introduced bugs using advanced techniques with explicit deployment time
func (d *DeploymentBugDetector) AnalyzeWithDeploymentTime(ctx context.Context, serviceName string, deploymentTime time.Time) (*Detection, error) {
	logger.Info("Starting deployment bug analysis",
		zap.String("service", serviceName),
		zap.Time("deployment_time", deploymentTime),
	)

	// Analyze time windows before and after deployment
	preWindow := 15 * time.Minute
	postWindow := 15 * time.Minute

	confidence := 0.0
	evidence := make(map[string]interface{})

	// 1. Error rate change point detection
	errorRateChange, errorSignificant := d.detectErrorRateChange(ctx, serviceName, deploymentTime, preWindow, postWindow)
	if errorSignificant {
		confidence += 35.0
		evidence["error_rate_spike"] = true
		evidence["error_rate_change_percent"] = fmt.Sprintf("%.1f", errorRateChange)
	}

	// 2. Response time change point detection
	responseChange, responseSignificant := d.detectResponseTimeChange(ctx, serviceName, deploymentTime, preWindow, postWindow)
	if responseSignificant {
		confidence += 30.0
		evidence["response_time_degradation"] = true
		evidence["response_time_change_percent"] = fmt.Sprintf("%.1f", responseChange)
	}

	// 3. Resource usage anomaly detection
	cpuAnomaly, memoryAnomaly := d.detectResourceAnomalies(ctx, serviceName, deploymentTime, postWindow)
	if cpuAnomaly {
		confidence += 15.0
		evidence["cpu_anomaly_detected"] = true
	}
	if memoryAnomaly {
		confidence += 15.0
		evidence["memory_anomaly_detected"] = true
	}

	// 4. Request success rate analysis
	successRateDrop := d.analyzeSuccessRate(ctx, serviceName, deploymentTime, preWindow, postWindow)
	if successRateDrop > 5.0 {
		confidence += 20.0
		evidence["success_rate_drop_percent"] = fmt.Sprintf("%.1f", successRateDrop)
	}

	// 5. Statistical significance test (Z-score)
	zScore := d.calculateZScore(ctx, errorRateChange, responseChange)
	if zScore > 2.0 {
		confidence += 10.0
		evidence["statistically_significant"] = true
		evidence["z_score"] = fmt.Sprintf("%.2f", zScore)
	}

	// 6. Timing correlation - degradation started right after deployment
	timingCorrelation := d.analyzeTimingCorrelation(ctx, serviceName, deploymentTime)
	if timingCorrelation > 0.8 {
		confidence += 15.0
		evidence["strong_timing_correlation"] = true
		evidence["correlation_score"] = fmt.Sprintf("%.2f", timingCorrelation)
	}

	evidence["deployment_time"] = deploymentTime.Format(time.RFC3339)
	evidence["analysis_window_min"] = 15

	detected := confidence > 75.0
	severity := d.calculateSeverity(confidence, errorRateChange, successRateDrop)
	recommendation := d.buildRecommendation(detected, severity, errorRateChange, responseChange, successRateDrop)

	logger.Info("Deployment bug analysis complete",
		zap.String("service", serviceName),
		zap.Bool("detected", detected),
		zap.Float64("confidence", confidence),
		zap.String("severity", severity),
	)

	return &Detection{
		Type:           DetectionDeploymentBug,
		ServiceName:    serviceName,
		Detected:       detected,
		Confidence:     confidence,
		Timestamp:      time.Now(),
		Evidence:       evidence,
		Recommendation: recommendation,
		Severity:       severity,
	}, nil
}

func (d *DeploymentBugDetector) detectErrorRateChange(ctx context.Context, serviceName string, deploymentTime time.Time, preWindow, postWindow time.Duration) (changePercent float64, significant bool) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return 0, false
	default:
	}

	// Get error rates before deployment
	preMetrics, err := d.db.GetMetricsInRange(serviceName, "error_rate", deploymentTime.Add(-preWindow), deploymentTime)
	if err != nil || len(preMetrics) < 3 {
		return 0, false
	}

	// Get error rates after deployment
	postMetrics, err := d.db.GetMetricsInRange(serviceName, "error_rate", deploymentTime, deploymentTime.Add(postWindow))
	if err != nil || len(postMetrics) < 3 {
		return 0, false
	}

	preAvg := CalculateAverageFromRecords(preMetrics)
	postAvg := CalculateAverageFromRecords(postMetrics)

	if preAvg == 0 {
		preAvg = 0.01 // Avoid division by zero
	}

	changePercent = ((postAvg - preAvg) / preAvg) * 100

	// Significant if error rate increased by > 50% AND absolute increase > 1%
	significant = changePercent > 50.0 && (postAvg-preAvg) > 1.0

	return changePercent, significant
}

func (d *DeploymentBugDetector) detectResponseTimeChange(ctx context.Context, serviceName string, deploymentTime time.Time, preWindow, postWindow time.Duration) (changePercent float64, significant bool) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return 0, false
	default:
	}

	// Get response times before deployment
	preMetrics, err := d.db.GetMetricsInRange(serviceName, "response_time", deploymentTime.Add(-preWindow), deploymentTime)
	if err != nil || len(preMetrics) < 3 {
		return 0, false
	}

	// Get response times after deployment
	postMetrics, err := d.db.GetMetricsInRange(serviceName, "response_time", deploymentTime, deploymentTime.Add(postWindow))
	if err != nil || len(postMetrics) < 3 {
		return 0, false
	}

	preAvg := CalculateAverageFromRecords(preMetrics)
	postAvg := CalculateAverageFromRecords(postMetrics)

	if preAvg == 0 {
		preAvg = 1.0 // Avoid division by zero
	}

	changePercent = ((postAvg - preAvg) / preAvg) * 100

	// Significant if response time increased by > 30% AND absolute increase > 100ms
	significant = changePercent > 30.0 && (postAvg-preAvg) > 100.0

	return changePercent, significant
}

func (d *DeploymentBugDetector) detectResourceAnomalies(ctx context.Context, serviceName string, deploymentTime time.Time, postWindow time.Duration) (cpuAnomaly, memoryAnomaly bool) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return false, false
	default:
	}

	// Get CPU usage after deployment
	cpuMetrics, err := d.db.GetMetricsInRange(serviceName, "cpu_usage", deploymentTime, deploymentTime.Add(postWindow))
	if err == nil && len(cpuMetrics) >= 5 {
		cpuValues := RecordsToValues(cpuMetrics)
		cpuAnomaly = HasAnomalies(cpuValues, 2.5) // Z-score threshold
	}

	// Check memory usage anomalies after deployment
	memMetrics, err := d.db.GetMetricsInRange(serviceName, "memory_usage", deploymentTime, deploymentTime.Add(postWindow))
	if err == nil && len(memMetrics) >= 5 {
		memValues := RecordsToValues(memMetrics)
		memoryAnomaly = HasAnomalies(memValues, 2.5)
	}

	return cpuAnomaly, memoryAnomaly
}

func (d *DeploymentBugDetector) analyzeSuccessRate(ctx context.Context, serviceName string, deploymentTime time.Time, preWindow, postWindow time.Duration) float64 {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return 0
	default:
	}

	// Get success rate before deployment
	preMetrics, err := d.db.GetMetricsInRange(serviceName, "success_rate", deploymentTime.Add(-preWindow), deploymentTime)
	if err != nil || len(preMetrics) < 3 {
		return 0
	}

	// Get success rate after deployment
	postMetrics, err := d.db.GetMetricsInRange(serviceName, "success_rate", deploymentTime, deploymentTime.Add(postWindow))
	if err != nil || len(postMetrics) < 3 {
		return 0
	}

	preAvg := CalculateAverageFromRecords(preMetrics)
	postAvg := CalculateAverageFromRecords(postMetrics)

	// Return absolute drop in success rate (e.g., 98% -> 93% = 5% drop)
	return math.Max(0, preAvg-postAvg)
}

func (d *DeploymentBugDetector) calculateZScore(ctx context.Context, errorChange, responseChange float64) float64 {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return 0
	default:
	}

	// Simple Z-score calculation based on changes
	// Normalized to common scale
	normalizedError := errorChange / 100.0
	normalizedResponse := responseChange / 100.0

	combinedChange := (normalizedError + normalizedResponse) / 2.0
	return math.Abs(combinedChange) * 3.0
}

func (d *DeploymentBugDetector) analyzeTimingCorrelation(ctx context.Context, serviceName string, deploymentTime time.Time) float64 {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return 0
	default:
	}

	// Check if degradation started within 2 minutes of deployment
	checkWindow := 10 * time.Minute

	errorMetrics, err := d.db.GetMetricsInRange(serviceName, "error_rate", deploymentTime, deploymentTime.Add(checkWindow))
	if err != nil || len(errorMetrics) < 2 {
		return 0
	}

	// Find first spike after deployment
	firstSpike := deploymentTime
	threshold := CalculateAverageFromRecords(errorMetrics) * 1.5

	for _, m := range errorMetrics {
		if m.Value > threshold {
			firstSpike = m.Timestamp
			break
		}
	}

	// Calculate correlation based on time proximity
	timeDiff := firstSpike.Sub(deploymentTime).Minutes()
	if timeDiff < 2.0 {
		return 1.0 // Perfect correlation
	} else if timeDiff < 5.0 {
		return 0.8 // Strong correlation
	} else if timeDiff < 10.0 {
		return 0.5 // Moderate correlation
	}

	return 0.2 // Weak correlation
}

func (d *DeploymentBugDetector) buildRecommendation(detected bool, severity string, errorChange, responseChange, successDrop float64) string {
	if !detected {
		return "No deployment-related issues detected. Metrics appear stable after deployment."
	}

	recommendation := ""

	switch severity {
	case "CRITICAL":
		recommendation = "CRITICAL DEPLOYMENT BUG: Immediate rollback recommended. "
	case "HIGH":
		recommendation = "HIGH PRIORITY DEPLOYMENT ISSUE: Consider rollback. "
	default:
		recommendation = "DEPLOYMENT WARNING: Monitor closely. "
	}

	if errorChange > 100.0 {
		recommendation += fmt.Sprintf("Error rate increased by %.0f%%. ", errorChange)
	}
	if responseChange > 50.0 {
		recommendation += fmt.Sprintf("Response time degraded by %.0f%%. ", responseChange)
	}
	if successDrop > 5.0 {
		recommendation += fmt.Sprintf("Success rate dropped by %.1f%%. ", successDrop)
	}

	recommendation += "Actions: 1) Review deployment diff and recent code changes. 2) Check application logs for new errors. 3) Verify database migrations completed successfully. 4) Consider rolling back to previous version."

	return recommendation
}

func (d *DeploymentBugDetector) calculateSeverity(confidence, errorChange, successDrop float64) string {
	if confidence < 75 {
		return "LOW"
	}

	// CRITICAL: Major error spike or success rate crash
	if errorChange > 200.0 || successDrop > 10.0 {
		return "CRITICAL"
	}

	// HIGH: Significant degradation
	if errorChange > 100.0 || successDrop > 5.0 {
		return "HIGH"
	}

	// MEDIUM: Moderate issues
	if errorChange > 50.0 || successDrop > 2.0 {
		return "MEDIUM"
	}

	return "LOW"
}
