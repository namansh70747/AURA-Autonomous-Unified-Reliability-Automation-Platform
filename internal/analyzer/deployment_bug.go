package analyzer

import (
	"context"
	"fmt"
	"time"

	"github.com/namansh70747/AURA-Autonomous-Unified-Reliability-Automation-Platform/internal/storage"
	"github.com/namansh70747/AURA-Autonomous-Unified-Reliability-Automation-Platform/pkg/logger"
	"go.uber.org/zap"
)

type DeploymentBugDetector struct {
	db *storage.PostgresClient
}

func NewDeploymentBugDetector(db *storage.PostgresClient) *DeploymentBugDetector {
	return &DeploymentBugDetector{
		db: db,
	}
}

func (d *DeploymentBugDetector) Analyze(ctx context.Context, serviceName string) (*Detection, error) {
	errorMetricNames := []string{"app_errors_total", "error_rate", "errors_total", "error_count"}

	var errorMetrics []*storage.Metric
	var err error

	for _, name := range errorMetricNames {
		errorMetrics, err = d.db.GetRecentMetrics(ctx, serviceName, name, 20*time.Minute)
		if err == nil && len(errorMetrics) >= 5 {
			break
		}
	}

	if err != nil || len(errorMetrics) < 5 {
		logger.Debug("Insufficient error data for deployment bug detection",
			zap.String("service", serviceName),
			zap.Int("data_points", len(errorMetrics)),
		)
		return &Detection{
			Type:        DetectionDeploymentBug,
			ServiceName: serviceName,
			Detected:    false,
			Confidence:  0,
			Timestamp:   time.Now(),
			Evidence: map[string]interface{}{
				"reason": "insufficient error data",
				"points": len(errorMetrics),
			},
			Recommendation: "Waiting for error rate data",
			Severity:       "LOW",
		}, nil
	}


	midPoint := len(errorMetrics) / 2
	beforeMetrics := errorMetrics[:midPoint]
	afterMetrics := errorMetrics[midPoint:]

	errorsBefore := d.calculateAverage(beforeMetrics)
	errorsAfter := d.calculateAverage(afterMetrics)

	if errorsBefore == 0 {
		errorsBefore = 0.1
	}

	errorIncrease := ((errorsAfter - errorsBefore) / errorsBefore) * 100

	cpuNormal := d.checkResourceNormal(ctx, serviceName, "cpu_usage_percent", 70.0)
	if !cpuNormal {
		cpuNormal = d.checkResourceNormal(ctx, serviceName, "cpu_usage", 70.0)
	}

	memoryNormal := d.checkResourceNormal(ctx, serviceName, "memory_usage_percent", 80.0)
	if !memoryNormal {
		memoryNormal = d.checkResourceNormal(ctx, serviceName, "memory_usage", 80.0)
	}

	confidence := 0.0

	if errorIncrease > 500 {
		confidence += 70.0
	} else if errorIncrease > 200 {
		confidence += 50.0
	} else if errorIncrease > 100 {
		confidence += 30.0
	}

	if cpuNormal && memoryNormal {
		confidence += 20.0
	}

	if errorsAfter > 15.0 {
		confidence += 10.0
	}

	detected := confidence > 80.0
	severity := d.calculateSeverity(confidence, errorsAfter)

	recommendation := "No action needed"
	if detected {
		if errorsAfter > 20.0 {
			recommendation = "CRITICAL: Rollback deployment immediately. Error rate critical."
		} else if errorsAfter > 10.0 {
			recommendation = "HIGH: Consider rollback. Investigate errors in new code."
		} else {
			recommendation = "MEDIUM: Monitor closely. New deployment may have introduced bugs."
		}
	}

	logger.Info("Deployment bug analysis complete",
		zap.String("service", serviceName),
		zap.Bool("detected", detected),
		zap.Float64("confidence", confidence),
	)

	return &Detection{
		Type:        DetectionDeploymentBug,
		ServiceName: serviceName,
		Detected:    detected,
		Confidence:  confidence,
		Timestamp:   time.Now(),
		Evidence: map[string]interface{}{
			"error_rate_before": fmt.Sprintf("%.2f%%", errorsBefore),
			"error_rate_after":  fmt.Sprintf("%.2f%%", errorsAfter),
			"error_increase":    fmt.Sprintf("%.0f%%", errorIncrease),
			"cpu_normal":        cpuNormal,
			"memory_normal":     memoryNormal,
			"data_points":       len(errorMetrics),
		},
		Recommendation: recommendation,
		Severity:       severity,
	}, nil
}

func (d *DeploymentBugDetector) calculateAverage(metrics []*storage.Metric) float64 {
	if len(metrics) == 0 {
		return 0
	}

	var sum float64
	for _, m := range metrics {
		sum += m.MetricValue
	}

	return sum / float64(len(metrics))
}

func (d *DeploymentBugDetector) checkResourceNormal(ctx context.Context, serviceName, metricType string, threshold float64) bool {
	metric, err := d.db.GetLatestMetric(ctx, serviceName, metricType)
	if err != nil || metric == nil {
		return true // Assume normal if no data
	}

	return metric.MetricValue < threshold
}

func (d *DeploymentBugDetector) calculateSeverity(confidence, errorRate float64) string {
	if confidence < 80 {
		return "LOW"
	}

	if errorRate > 20 {
		return "CRITICAL"
	} else if errorRate > 10 {
		return "HIGH"
	} else if errorRate > 5 {
		return "MEDIUM"
	}

	return "LOW"
}
