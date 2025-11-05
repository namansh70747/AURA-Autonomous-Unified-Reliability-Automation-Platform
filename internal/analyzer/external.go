package analyzer

import (
	"context"
	"fmt"
	"time"

	"github.com/namansh70747/AURA-Autonomous-Unified-Reliability-Automation-Platform/internal/storage"
	"github.com/namansh70747/AURA-Autonomous-Unified-Reliability-Automation-Platform/pkg/logger"
	"go.uber.org/zap"
)

type ExternalFailureDetector struct {
	db *storage.PostgresClient
}

func NewExternalFailureDetector(db *storage.PostgresClient) *ExternalFailureDetector {
	return &ExternalFailureDetector{
		db: db,
	}
}

func (e *ExternalFailureDetector) Analyze(ctx context.Context, serviceName string) (*Detection, error) {

	cpuMetric, _ := e.db.GetLatestMetric(ctx, serviceName, "cpu_usage_percent")
	if cpuMetric == nil {
		cpuMetric, _ = e.db.GetLatestMetric(ctx, serviceName, "cpu_usage")
	}

	memoryMetric, _ := e.db.GetLatestMetric(ctx, serviceName, "memory_usage_percent")
	if memoryMetric == nil {
		memoryMetric, _ = e.db.GetLatestMetric(ctx, serviceName, "memory_usage")
	}

	errorMetricNames := []string{"app_errors_total", "error_rate", "error_count"}
	var errorMetrics []*storage.Metric
	var err error

	for _, name := range errorMetricNames {
		errorMetrics, err = e.db.GetRecentMetrics(ctx, serviceName, name, 5*time.Minute)
		if err == nil && len(errorMetrics) > 0 {
			break
		}
	}

	if err != nil || len(errorMetrics) == 0 {
		logger.Debug("Insufficient error data for external failure detection",
			zap.String("service", serviceName),
		)
		return &Detection{
			Type:        DetectionExternalFailure,
			ServiceName: serviceName,
			Detected:    false,
			Confidence:  0,
			Timestamp:   time.Now(),
			Evidence: map[string]interface{}{
				"reason": "insufficient error data",
			},
			Recommendation: "Waiting for error rate data",
			Severity:       "LOW",
		}, nil
	}

	currentErrorRate := errorMetrics[len(errorMetrics)-1].MetricValue

	resourcesNormal := true
	cpuUsage := 0.0
	memoryUsage := 0.0

	if cpuMetric != nil {
		cpuUsage = cpuMetric.MetricValue
		if cpuUsage > 70 {
			resourcesNormal = false
		}
	}

	if memoryMetric != nil {
		memoryUsage = memoryMetric.MetricValue
		if memoryUsage > 80 {
			resourcesNormal = false
		}
	}

	confidence := 0.0
	if currentErrorRate > 15.0 {
		confidence += 40.0
	} else if currentErrorRate > 10.0 {
		confidence += 25.0
	}

	if resourcesNormal {
		confidence += 40.0
	}

	if len(errorMetrics) > 3 {
		recentErrors := errorMetrics[len(errorMetrics)-3:]
		allHigh := true
		for _, m := range recentErrors {
			if m.MetricValue < 10.0 {
				allHigh = false
				break
			}
		}
		if allHigh {
			confidence += 10.0
		}
	}

	detected := confidence > 80.0
	severity := e.calculateSeverity(confidence, currentErrorRate)

	recommendation := "No action needed"
	if detected {
		recommendation = fmt.Sprintf(
			"External dependency failure suspected. Our resources normal (CPU: %.1f%%, Memory: %.1f%%). "+
				"Check downstream services/APIs. Consider enabling circuit breaker.",
			cpuUsage, memoryUsage,
		)
	}

	logger.Info("External failure analysis complete",
		zap.String("service", serviceName),
		zap.Bool("detected", detected),
		zap.Float64("confidence", confidence),
	)

	return &Detection{
		Type:        DetectionExternalFailure,
		ServiceName: serviceName,
		Detected:    detected,
		Confidence:  confidence,
		Timestamp:   time.Now(),
		Evidence: map[string]interface{}{
			"error_rate":       fmt.Sprintf("%.2f%%", currentErrorRate),
			"cpu_usage":        fmt.Sprintf("%.1f%%", cpuUsage),
			"memory_usage":     fmt.Sprintf("%.1f%%", memoryUsage),
			"resources_normal": resourcesNormal,
		},
		Recommendation: recommendation,
		Severity:       severity,
	}, nil
}

func (e *ExternalFailureDetector) calculateSeverity(confidence, errorRate float64) string {
    if confidence < 80 {
        return "LOW"
    }

    if errorRate > 25 {
        return "CRITICAL"
    } else if errorRate > 15 {
        return "HIGH"
    } else if errorRate > 10 {
        return "MEDIUM"
    }

    return "LOW"
}