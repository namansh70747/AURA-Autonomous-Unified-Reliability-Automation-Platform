package analyzer

import (
	"context"
	"time"

	"github.com/namansh70747/AURA-Autonomous-Unified-Reliability-Automation-Platform/internal/storage"
	"github.com/namansh70747/AURA-Autonomous-Unified-Reliability-Automation-Platform/pkg/logger"
	"go.uber.org/zap"
)

type CascadeDetector struct {
	db *storage.PostgresClient
}

func NewCascadeDetector(db *storage.PostgresClient) *CascadeDetector {
	return &CascadeDetector{
		db: db,
	}
}

func (c *CascadeDetector) Analyze(ctx context.Context, serviceName string) (*Detection, error) {
	latencyMetrics, err := c.db.GetRecentMetrics(ctx, serviceName, "http_latency", 10*time.Minute)
	if err != nil || len(latencyMetrics) < 5 {
		logger.Debug("Insufficient latency data for cascade detection",
			zap.String("service", serviceName),
		)
		return &Detection{
			Type:        DetectionCascadingFailure,
			ServiceName: serviceName,
			Detected:    false,
			Confidence:  0,
			Timestamp:   time.Now(),
			Evidence: map[string]interface{}{
				"reason": "insufficient latency data for cascade detection",
			},
			Recommendation: "Cascade detection requires dependency mapping (Phase 2.5)",
			Severity:       "LOW",
		}, nil
	}

	currentLatency := latencyMetrics[len(latencyMetrics)-1].MetricValue
	avgLatency := c.calculateAverage(latencyMetrics)

	confidence := 0.0

	if currentLatency > avgLatency*2 && currentLatency > 1000 {
		confidence = 60.0
	}

	logger.Debug("Cascade analysis complete (simplified)",
		zap.String("service", serviceName),
		zap.Float64("confidence", confidence),
	)

	return &Detection{
		Type:        DetectionCascadingFailure,
		ServiceName: serviceName,
		Detected:    confidence > 50.0,
		Confidence:  confidence,
		Timestamp:   time.Now(),
		Evidence: map[string]interface{}{
			"current_latency_ms": currentLatency,
			"average_latency_ms": avgLatency,
			"note":               "Full cascade detection in Phase 2.5",
		},
		Recommendation: "Monitor dependent services for failures",
		Severity:       "MEDIUM",
	}, nil
}

func (c *CascadeDetector) calculateAverage(metrics []*storage.Metric) float64 {
	if len(metrics) == 0 {
		return 0
	}

	var sum float64
	for _, m := range metrics {
		sum += m.MetricValue
	}

	return sum / float64(len(metrics))
}
