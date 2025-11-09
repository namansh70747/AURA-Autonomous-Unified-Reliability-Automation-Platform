package analyzer

import (
	"context"
	"fmt"
	"time"

	"github.com/namansh70747/AURA-Autonomous-Unified-Reliability-Automation-Platform/internal/storage"
	"github.com/namansh70747/AURA-Autonomous-Unified-Reliability-Automation-Platform/pkg/logger"
	"go.uber.org/zap"
)

type CascadeDetector struct {
	db         *storage.PostgresClient
	correlator *ServiceCorrelator
}

func NewCascadeDetector(db *storage.PostgresClient) *CascadeDetector {
	return &CascadeDetector{
		db:         db,
		correlator: NewServiceCorrelator(db),
	}
}

func (c *CascadeDetector) Analyze(ctx context.Context, serviceName string) (*Detection, error) {
	logger.Info("Starting cascade failure analysis",
		zap.String("service", serviceName),
	)

	// Step 1: Get latency metrics
	latencyMetrics, err := c.db.GetRecentMetrics(ctx, serviceName, "http_latency", 15*time.Minute)
	if err != nil || len(latencyMetrics) < 5 {
		// Try alternative latency metric names
		latencyMetrics, err = c.db.GetRecentMetrics(ctx, serviceName, "response_time", 15*time.Minute)
		if err != nil || len(latencyMetrics) < 5 {
			latencyMetrics, err = c.db.GetRecentMetrics(ctx, serviceName, "latency_ms", 15*time.Minute)
		}
	}

	if err != nil || len(latencyMetrics) < 5 {
		logger.Debug("Insufficient latency data for cascade detection",
			zap.String("service", serviceName),
			zap.Int("data_points", len(latencyMetrics)),
		)
		return &Detection{
			Type:        DetectionCascadingFailure,
			ServiceName: serviceName,
			Detected:    false,
			Confidence:  0,
			Timestamp:   time.Now(),
			Evidence: map[string]interface{}{
				"reason":      "insufficient latency data",
				"data_points": len(latencyMetrics),
			},
			Recommendation: "Need more latency data for cascade analysis",
			Severity:       "LOW",
		}, nil
	}

	// Step 2: Analyze latency pattern
	currentLatency := latencyMetrics[len(latencyMetrics)-1].MetricValue
	avgLatency := CalculateAverage(latencyMetrics)
	maxLatency := CalculateMax(latencyMetrics)
	latencyVolatility := CalculateVolatility(latencyMetrics)

	// Check for sudden latency spike
	latencySpike := currentLatency > avgLatency*2.0 && currentLatency > 500 // > 500ms

	confidence := 0.0
	evidence := make(map[string]interface{})

	// Step 3: Latency spike analysis (30 points)
	if latencySpike {
		confidence += 30.0
		spikeIntensity := (currentLatency - avgLatency) / avgLatency * 100
		evidence["latency_spike"] = true
		evidence["spike_intensity_percent"] = fmt.Sprintf("%.0f", spikeIntensity)

		logger.Debug("Latency spike detected",
			zap.String("service", serviceName),
			zap.Float64("current", currentLatency),
			zap.Float64("average", avgLatency),
		)
	}

	// Step 4: Get error rate metrics
	errorMetrics, err := c.getErrorMetrics(ctx, serviceName)
	errorRateIncreasing := false
	currentErrorRate := 0.0

	if err == nil && len(errorMetrics) >= 5 {
		currentErrorRate = errorMetrics[len(errorMetrics)-1].MetricValue
		mid := len(errorMetrics) / 2
		firstHalf := CalculateAverage(errorMetrics[:mid])
		secondHalf := CalculateAverage(errorMetrics[mid:])

		errorRateIncreasing = secondHalf > firstHalf*1.5 && currentErrorRate > 5.0

		if errorRateIncreasing {
			confidence += 20.0
			evidence["error_rate_increasing"] = true
			evidence["current_error_rate"] = fmt.Sprintf("%.2f%%", currentErrorRate)
		}
	}

	// Step 5: Analyze service correlations (Phase 2.5 feature)
	relatedServices, cascadeRisk := c.analyzeServiceCorrelations(ctx, serviceName)

	if cascadeRisk > 60.0 {
		confidence += 25.0
		evidence["correlated_failures"] = true
		evidence["affected_services"] = relatedServices
		evidence["cascade_risk_score"] = cascadeRisk

		logger.Info("Correlated service failures detected",
			zap.String("source_service", serviceName),
			zap.Strings("affected_services", relatedServices),
			zap.Float64("risk_score", cascadeRisk),
		)
	}

	// Step 6: Check for upstream dependency issues
	upstreamIssue := c.detectUpstreamIssue(ctx, serviceName)
	if upstreamIssue {
		confidence += 15.0
		evidence["upstream_dependency_issue"] = true
	}

	// Step 7: Time-pattern analysis (look for propagation delay)
	propagationDetected := c.detectPropagationPattern(latencyMetrics, errorMetrics)
	if propagationDetected {
		confidence += 10.0
		evidence["propagation_pattern"] = true
		evidence["propagation_note"] = "Failure appears to be propagating from another service"
	}

	// Calculate final metrics
	detected := confidence > 80.0
	severity := c.calculateSeverity(confidence, currentLatency, currentErrorRate, len(relatedServices))

	// Build recommendation
	recommendation := c.buildRecommendation(
		detected,
		severity,
		relatedServices,
		currentLatency,
		avgLatency,
		currentErrorRate,
		upstreamIssue,
	)

	// Add all evidence
	evidence["current_latency_ms"] = currentLatency
	evidence["average_latency_ms"] = avgLatency
	evidence["max_latency_ms"] = maxLatency
	evidence["latency_volatility"] = latencyVolatility
	evidence["data_points"] = len(latencyMetrics)

	logger.Info("Cascade failure analysis complete",
		zap.String("service", serviceName),
		zap.Bool("detected", detected),
		zap.Float64("confidence", confidence),
		zap.String("severity", severity),
		zap.Int("affected_services", len(relatedServices)),
	)

	return &Detection{
		Type:           DetectionCascadingFailure,
		ServiceName:    serviceName,
		Detected:       detected,
		Confidence:     confidence,
		Timestamp:      time.Now(),
		Evidence:       evidence,
		Recommendation: recommendation,
		Severity:       severity,
	}, nil
}

// analyzeServiceCorrelations finds services with correlated failures
func (c *CascadeDetector) analyzeServiceCorrelations(ctx context.Context, serviceName string) ([]string, float64) {
	// Get all services in the system
	allServices, err := c.db.GetAllServices(ctx)
	if err != nil || len(allServices) <= 1 {
		return []string{}, 0
	}

	affectedServices := []string{}
	totalRisk := 0.0
	correlationCount := 0

	for _, otherService := range allServices {
		if otherService == serviceName {
			continue
		}

		// Check error rate correlation
		result, err := c.correlator.CalculatePearsonCorrelation(
			serviceName, "error_rate",
			otherService, "error_rate",
			10*time.Minute,
		)

		if err != nil {
			continue
		}

		// Strong positive correlation indicates cascade
		if result.Correlation > 0.6 && result.Strength != "insufficient_data" {
			affectedServices = append(affectedServices, otherService)
			totalRisk += result.CascadeRisk
			correlationCount++

			logger.Debug("Correlated service found",
				zap.String("source", serviceName),
				zap.String("affected", otherService),
				zap.Float64("correlation", result.Correlation),
			)
		}
	}

	avgRisk := 0.0
	if correlationCount > 0 {
		avgRisk = totalRisk / float64(correlationCount)
	}

	return affectedServices, avgRisk
}

// detectUpstreamIssue checks if the problem originates from a dependency
func (c *CascadeDetector) detectUpstreamIssue(ctx context.Context, serviceName string) bool {
	// Check if our service's CPU/Memory is normal while errors are high
	cpuMetric, _ := c.db.GetLatestMetric(ctx, serviceName, "cpu_usage")
	if cpuMetric == nil {
		cpuMetric, _ = c.db.GetLatestMetric(ctx, serviceName, "cpu_usage_percent")
	}

	memoryMetric, _ := c.db.GetLatestMetric(ctx, serviceName, "memory_usage")
	if memoryMetric == nil {
		memoryMetric, _ = c.db.GetLatestMetric(ctx, serviceName, "memory_usage_percent")
	}

	// If resources are normal but errors/latency are high, likely upstream issue
	resourcesNormal := true
	if cpuMetric != nil && cpuMetric.MetricValue > 70.0 {
		resourcesNormal = false
	}
	if memoryMetric != nil && memoryMetric.MetricValue > 80.0 {
		resourcesNormal = false
	}

	errorMetrics, err := c.getErrorMetrics(ctx, serviceName)
	if err != nil || len(errorMetrics) == 0 {
		return false
	}

	currentErrorRate := errorMetrics[len(errorMetrics)-1].MetricValue
	errorsHigh := currentErrorRate > 10.0

	return resourcesNormal && errorsHigh
}

// detectPropagationPattern looks for time-delayed correlation pattern
func (c *CascadeDetector) detectPropagationPattern(latencyMetrics, errorMetrics []*storage.Metric) bool {
	if len(latencyMetrics) < 5 || len(errorMetrics) < 5 {
		return false
	}

	// Check if latency increased first, then errors followed
	// This suggests downstream cascade
	latencyMid := len(latencyMetrics) / 2
	errorMid := len(errorMetrics) / 2

	earlyLatency := CalculateAverage(latencyMetrics[:latencyMid])
	lateLatency := CalculateAverage(latencyMetrics[latencyMid:])

	earlyErrors := CalculateAverage(errorMetrics[:errorMid])
	lateErrors := CalculateAverage(errorMetrics[errorMid:])

	latencyIncreased := lateLatency > earlyLatency*1.5
	errorsIncreased := lateErrors > earlyErrors*1.5

	return latencyIncreased && errorsIncreased
}

// getErrorMetrics tries multiple error metric names
func (c *CascadeDetector) getErrorMetrics(ctx context.Context, serviceName string) ([]*storage.Metric, error) {
	errorMetricNames := []string{"error_rate", "app_errors_total", "errors_total", "error_count"}

	for _, metricName := range errorMetricNames {
		metrics, err := c.db.GetRecentMetrics(ctx, serviceName, metricName, 10*time.Minute)
		if err == nil && len(metrics) > 0 {
			return metrics, nil
		}
	}

	return nil, fmt.Errorf("no error metrics found")
}

// buildRecommendation creates detailed recommendation
func (c *CascadeDetector) buildRecommendation(
	detected bool,
	severity string,
	affectedServices []string,
	currentLatency, avgLatency, errorRate float64,
	upstreamIssue bool,
) string {
	if !detected {
		return "No cascading failure detected. Service operating normally."
	}

	recommendation := ""

	switch severity {
	case "CRITICAL":
		recommendation = "CRITICAL CASCADE FAILURE: Immediate action required. "
	case "HIGH":
		recommendation = "HIGH PRIORITY: Cascading failure in progress. "
	default:
		recommendation = "CASCADE WARNING: "
	}

	// Add specific actions
	if len(affectedServices) > 0 {
		recommendation += fmt.Sprintf(
			"Cascade affecting %d services (%v). ",
			len(affectedServices),
			affectedServices,
		)
		recommendation += "1) Enable circuit breakers. 2) Isolate failing service. "
	}

	if upstreamIssue {
		recommendation += "Root cause likely in upstream dependency - check external APIs/databases. "
	}

	if currentLatency > avgLatency*3 {
		recommendation += fmt.Sprintf(
			"Latency critical (%.0fms vs normal %.0fms). Consider scaling immediately. ",
			currentLatency,
			avgLatency,
		)
	}

	if errorRate > 20 {
		recommendation += "High error rate detected - implement fallback mechanisms. "
	}

	recommendation += "Monitor all dependent services closely."

	return recommendation
}

// calculateSeverity determines severity based on multiple factors
func (c *CascadeDetector) calculateSeverity(
	confidence, latency, errorRate float64,
	affectedServiceCount int,
) string {
	if confidence < 80 {
		return "LOW"
	}

	// CRITICAL: Multiple services affected + high errors + high latency
	if affectedServiceCount >= 3 || errorRate > 20 || latency > 5000 {
		return "CRITICAL"
	}

	// HIGH: Some services affected or very high latency/errors
	if affectedServiceCount >= 2 || errorRate > 15 || latency > 3000 {
		return "HIGH"
	}

	// MEDIUM: One service affected or moderate issues
	if affectedServiceCount >= 1 || errorRate > 10 || latency > 2000 {
		return "MEDIUM"
	}

	return "LOW"
}
