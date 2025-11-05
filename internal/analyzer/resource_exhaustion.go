package analyzer

import (
	"context"
	"fmt"
	"time"

	"github.com/namansh70747/AURA-Autonomous-Unified-Reliability-Automation-Platform/internal/storage"
	"github.com/namansh70747/AURA-Autonomous-Unified-Reliability-Automation-Platform/pkg/logger"
	"go.uber.org/zap"
)

type ResourceExhaustionDetector struct {
	db *storage.PostgresClient
}

func NewResourceExhaustionDetector(db *storage.PostgresClient) *ResourceExhaustionDetector {
	return &ResourceExhaustionDetector{
		db: db,
	}
}

func (r *ResourceExhaustionDetector) Analyze(ctx context.Context, serviceName string) (*Detection, error) {
	// Get CPU metrics
	cpuMetric, _ := r.db.GetLatestMetric(ctx, serviceName, "cpu_usage")
	if cpuMetric == nil {
		cpuMetric, _ = r.db.GetLatestMetric(ctx, serviceName, "cpu_usage_percent")
	}

	// Get memory metrics
	memoryMetric, _ := r.db.GetLatestMetric(ctx, serviceName, "memory_usage")
	if memoryMetric == nil {
		memoryMetric, _ = r.db.GetLatestMetric(ctx, serviceName, "memory_usage_percent")
	}

	// Get error metrics
	errorMetric, _ := r.db.GetLatestMetric(ctx, serviceName, "error_count")
	if errorMetric == nil {
		errorMetric, _ = r.db.GetLatestMetric(ctx, serviceName, "error_rate")
	}

	if cpuMetric == nil && memoryMetric == nil {
		logger.Debug("No resource metrics available",
			zap.String("service", serviceName),
		)
		return &Detection{
			Type:        DetectionResourceExhaustion,
			ServiceName: serviceName,
			Detected:    false,
			Confidence:  0,
			Timestamp:   time.Now(),
			Evidence: map[string]interface{}{
				"reason": "no resource data available",
			},
			Recommendation: "Waiting for resource metrics",
			Severity:       "LOW",
		}, nil
	}

	cpuUsage := 0.0
	memoryUsage := 0.0
	errorRate := 0.0

	cpuHigh := false
	memoryHigh := false
	errorsHigh := false

	if cpuMetric != nil {
		cpuUsage = cpuMetric.MetricValue
		cpuHigh = cpuUsage > 85.0
	}

	if memoryMetric != nil {
		memoryUsage = memoryMetric.MetricValue
		memoryHigh = memoryUsage > 80.0
	}

	if errorMetric != nil {
		errorRate = errorMetric.MetricValue
		errorsHigh = errorRate > 10.0
	}

	// Calculate confidence
	confidence := 0.0

	if cpuHigh {
		confidence += 40.0
	}

	if memoryHigh {
		confidence += 40.0
	}

	if errorsHigh {
		confidence += 20.0
	}

	if cpuHigh && memoryHigh {
		confidence = 95.0
	}

	detected := confidence > 80.0
	severity := r.calculateSeverity(confidence, cpuUsage, memoryUsage, errorRate)

	recommendation := "No action needed"
	if detected {
		scaleFactor := r.calculateScaleFactor(cpuUsage, memoryUsage)

		recommendation = fmt.Sprintf(
			"Resource exhaustion detected. Current: CPU %.1f%%, Memory %.1f%%. "+
				"Recommended action: Scale up by %.0fx (add %d pods if current=5). "+
				"Or optimize resource limits.",
			cpuUsage, memoryUsage, scaleFactor, int(scaleFactor*5)-5,
		)
	}

	logger.Info("Resource exhaustion analysis complete",
		zap.String("service", serviceName),
		zap.Bool("detected", detected),
		zap.Float64("confidence", confidence),
		zap.String("severity", severity),
	)

	return &Detection{
		Type:        DetectionResourceExhaustion,
		ServiceName: serviceName,
		Detected:    detected,
		Confidence:  confidence,
		Timestamp:   time.Now(),
		Evidence: map[string]interface{}{
			"cpu_usage":    fmt.Sprintf("%.1f%%", cpuUsage),
			"memory_usage": fmt.Sprintf("%.1f%%", memoryUsage),
			"error_rate":   fmt.Sprintf("%.2f%%", errorRate),
			"cpu_high":     cpuHigh,
			"memory_high":  memoryHigh,
			"errors_high":  errorsHigh,
		},
		Recommendation: recommendation,
		Severity:       severity,
	}, nil
}

func (r *ResourceExhaustionDetector) calculateScaleFactor(cpuUsage, memoryUsage float64) float64 {
	maxUsage := cpuUsage
	if memoryUsage > maxUsage {
		maxUsage = memoryUsage
	}

	if maxUsage > 90 {
		return 2.0
	} else if maxUsage > 85 {
		return 1.5
	} else if maxUsage > 80 {
		return 1.3
	}

	return 1.2
}

func (r *ResourceExhaustionDetector) calculateSeverity(confidence, cpu, memory, errors float64) string {
	if confidence < 80 {
		return "LOW"
	}

	if cpu > 95 || memory > 95 || errors > 20 {
		return "CRITICAL"
	} else if cpu > 90 || memory > 90 || errors > 15 {
		return "HIGH"
	} else if cpu > 85 || memory > 85 {
		return "MEDIUM"
	}

	return "LOW"
}
