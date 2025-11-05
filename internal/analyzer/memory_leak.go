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

type MemoryLeakDetector struct {
	db *storage.PostgresClient
}

func NewMemoryLeakDetector(db *storage.PostgresClient) *MemoryLeakDetector {
	return &MemoryLeakDetector{
		db: db,
	}
}

func (m *MemoryLeakDetector) Analyze(ctx context.Context, serviceName string) (*Detection, error) {
	memoryMetrics, err := m.db.GetRecentMetrics(ctx, serviceName, "memory_usage", 1*time.Hour)
	if err != nil {
		logger.Debug("Failed to get memory metrics",
			zap.String("service", serviceName),
			zap.Error(err),
		)
		memoryMetrics, err = m.db.GetRecentMetrics(ctx, serviceName, "memory_usage_percent", 1*time.Hour)
	}

	if err != nil || len(memoryMetrics) < 10 {
		logger.Debug("Insufficient memory data",
			zap.String("service", serviceName),
			zap.Int("data_points", len(memoryMetrics)),
		)
		return &Detection{
			Type:        DetectionMemoryLeak,
			ServiceName: serviceName,
			Detected:    false,
			Confidence:  0,
			Timestamp:   time.Now(),
			Evidence: map[string]interface{}{
				"reason": "insufficient data",
				"points": len(memoryMetrics),
			},
			Recommendation: "Waiting for more data points (need 10+)",
			Severity:       "LOW",
		}, nil
	}

	// Step 2: Calculate growth rate using linear regression
	growthRate, r2 := m.calculateGrowthRate(memoryMetrics) //growth rate, growthRate = 30.0 (% per hour)|r2 = 0.98 (excellent fit)

	// Step 3: Check if traffic is stable
	trafficStable := m.checkTrafficStability(ctx, serviceName) //true or false

	// Step 4: Calculate confidence
	confidence := 0.0

	// Strong linear increase pattern
	if r2 > 0.90 && growthRate > 0 {
		confidence += 70.0
		logger.Debug("Strong linear pattern detected",
			zap.String("service", serviceName),
			zap.Float64("r2", r2),
			zap.Float64("growth_rate", growthRate),
		)
	} else if r2 > 0.70 && growthRate > 0 {
		confidence += 50.0
	}

	// Traffic is stable but memory growing
	if trafficStable {
		confidence += 15.0
		logger.Debug("Traffic stable while memory grows",
			zap.String("service", serviceName),
		)
	}

	// High growth rate
	if growthRate > 50.0 { // Growing more than 50 MB/hour
		confidence += 15.0
	}

	// Step 5: Calculate time to crash
	currentMemory := memoryMetrics[len(memoryMetrics)-1].MetricValue
	maxMemory := 90.0 // 90% threshold
	timeToExhaustion := "N/A"

	if growthRate > 0 && currentMemory < maxMemory {
		hoursRemaining := (maxMemory - currentMemory) / growthRate
		if hoursRemaining > 0 && hoursRemaining < 168 { // Less than 1 week
			timeToExhaustion = fmt.Sprintf("%.1f hours", hoursRemaining)
		}
	}

	detected := confidence > 80.0
	severity := m.calculateSeverity(confidence, growthRate, currentMemory)

	recommendation := "No action needed"
	if detected {
		if currentMemory > 80.0 {
			recommendation = "CRITICAL: Restart service immediately to prevent crash"
		} else if currentMemory > 70.0 {
			recommendation = "HIGH: Schedule service restart within 4 hours"
		} else {
			recommendation = "MEDIUM: Investigate memory leak and plan restart"
		}
	}

	logger.Info("Memory leak analysis complete",
		zap.String("service", serviceName),
		zap.Bool("detected", detected),
		zap.Float64("confidence", confidence),
		zap.String("severity", severity),
	)

	return &Detection{
		Type:        DetectionMemoryLeak,
		ServiceName: serviceName,
		Detected:    detected,
		Confidence:  confidence,
		Timestamp:   time.Now(),
		Evidence: map[string]interface{}{
			"growth_rate_mb_per_hour": growthRate,
			"current_memory_percent":  currentMemory,
			"r_squared":               r2,
			"traffic_stable":          trafficStable,
			"time_to_exhaustion":      timeToExhaustion,
			"data_points":             len(memoryMetrics),
		},
		Recommendation: recommendation,
		Severity:       severity,
	}, nil
}

func (m *MemoryLeakDetector) calculateGrowthRate(metrics []*storage.Metric) (growthRate float64, r2 float64) {
	if len(metrics) < 2 {
		return 0, 0
	}

	n := float64(len(metrics))

	var sumX, sumY, sumXY, sumX2, sumY2 float64

	for i, metric := range metrics {
		x := float64(i)
		y := metric.MetricValue

		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
		sumY2 += y * y
	}

	denominator := (n*sumX2 - sumX*sumX)
	if denominator == 0 {
		return 0, 0
	}

	slope := (n*sumXY - sumX*sumY) / denominator

	meanY := sumY / n
	var ssRes, ssTot float64

	for i, metric := range metrics {
		x := float64(i)
		y := metric.MetricValue
		intercept := (sumY - slope*sumX) / n
		predicted := (slope * x) + intercept

		ssRes += (y - predicted) * (y - predicted)
		ssTot += (y - meanY) * (y - meanY)
	}

	if ssTot == 0 {
		r2 = 0
	} else {
		r2 = 1 - (ssRes / ssTot)
	}

	timeSpan := metrics[len(metrics)-1].Timestamp.Sub(metrics[0].Timestamp).Hours()
	if timeSpan > 0 {
		growthRate = slope * (60.0 / timeSpan)
	}

	return math.Abs(growthRate), math.Abs(r2)
}

func (m *MemoryLeakDetector) checkTrafficStability(ctx context.Context, serviceName string) bool {
	metricNames := []string{"http_requests_total", "http_requests", "requests_total"}

	var metrics []*storage.Metric
	var err error

	for _, name := range metricNames {
		metrics, err = m.db.GetRecentMetrics(ctx, serviceName, name, 1*time.Hour)
		if err == nil && len(metrics) >= 10 {
			break
		}
	}

	if err != nil || len(metrics) < 10 {
		return false
	}

	var sum, sumSquares float64
	for _, metric := range metrics { //metric is each metric in metrics
		sum += metric.MetricValue                             //sum means sum of metricvalues
		sumSquares += metric.MetricValue * metric.MetricValue //sum of squares of metric value
	}

	n := float64(len(metrics)) //n is the length of metrics
	mean := sum / n            //average value
	variance := (sumSquares / n) - (mean * mean)
	stdDev := math.Sqrt(variance)

	if mean == 0 {
		return true
	}

	cv := stdDev / mean

	return cv < 0.3
} //check for sudden spikes in traffic and help us to calculate correctly

func (m *MemoryLeakDetector) calculateSeverity(confidence, growthRate, currentMemory float64) string {
	if confidence < 80 {
		return "LOW"
	}

	if currentMemory > 80 || growthRate > 100 {
		return "CRITICAL"
	} else if currentMemory > 70 || growthRate > 50 {
		return "HIGH"
	} else if currentMemory > 60 {
		return "MEDIUM"
	}

	return "LOW"
}
