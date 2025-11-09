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
	logger.Info("Starting memory leak analysis",
		zap.String("service", serviceName),
	)

	memoryMetrics, err := m.getMemoryMetrics(ctx, serviceName, 30*time.Minute)
	if err != nil || len(memoryMetrics) < 10 {
		logger.Debug("Insufficient memory data for leak detection",
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
				"reason":      "insufficient memory data",
				"data_points": len(memoryMetrics),
			},
			Recommendation: "Need at least 10 memory data points for analysis",
			Severity:       "LOW",
		}, nil
	}

	confidence := 0.0 // Confidence = 0
	evidence := make(map[string]interface{})

	slope, _, rSquared, growthRate := PerformLinearRegression(memoryMetrics)

	if slope > 0 && rSquared > 0.7 {
		confidence += 40.0
		evidence["memory_growth_detected"] = true
		evidence["growth_rate_mb_per_min"] = fmt.Sprintf("%.2f", slope*60)
		evidence["regression_r_squared"] = fmt.Sprintf("%.3f", rSquared)
	}

	trafficStable, trafficGrowth := m.analyzeTrafficPattern(ctx, serviceName)

	if !trafficStable && trafficGrowth > 50 {
		confidence = math.Max(0, confidence-30.0)
		evidence["traffic_spike"] = true
		evidence["traffic_growth_percent"] = trafficGrowth
	} else if trafficStable && slope > 0 {
		confidence += 25.0
		evidence["traffic_stable"] = true
		evidence["memory_growth_unexplained"] = true
	}

	accelerating := m.detectAcceleration(memoryMetrics)
	if accelerating {
		confidence += 15.0
		evidence["growth_accelerating"] = true
	}

	sustainedGrowth := m.verifySustainedGrowth(memoryMetrics)
	if sustainedGrowth {
		confidence += 10.0
		evidence["sustained_growth"] = true
	}

	volatility := CalculateVolatility(memoryMetrics)
	if volatility < 10.0 && slope > 0 {
		confidence += 10.0
		evidence["low_volatility"] = true
		evidence["volatility_percent"] = fmt.Sprintf("%.2f", volatility)
	}

	currentMemory := memoryMetrics[len(memoryMetrics)-1].MetricValue
	avgMemory := CalculateAverage(memoryMetrics)
	maxMemory := CalculateMax(memoryMetrics)
	minMemory := CalculateMin(memoryMetrics)
	memoryIncrease := ((currentMemory - minMemory) / minMemory) * 100

	evidence["current_memory_mb"] = currentMemory
	evidence["average_memory_mb"] = avgMemory
	evidence["max_memory_mb"] = maxMemory
	evidence["min_memory_mb"] = minMemory
	evidence["memory_increase_percent"] = fmt.Sprintf("%.1f", memoryIncrease)
	evidence["growth_rate_percent"] = fmt.Sprintf("%.2f", growthRate)
	evidence["data_points"] = len(memoryMetrics)

	detected := confidence > 80.0
	severity := m.calculateSeverity(confidence, growthRate, currentMemory)
	recommendation := m.buildRecommendation(detected, severity, growthRate, currentMemory, trafficStable, accelerating)

	return &Detection{
		Type:           DetectionMemoryLeak,
		ServiceName:    serviceName,
		Detected:       detected,
		Confidence:     confidence,
		Timestamp:      time.Now(),
		Evidence:       evidence,
		Recommendation: recommendation,
		Severity:       severity,
	}, nil
}

func (m *MemoryLeakDetector) analyzeTrafficPattern(ctx context.Context, serviceName string) (stable bool, growthPercent float64) {
	trafficMetrics, err := m.db.GetRecentMetrics(ctx, serviceName, "request_rate", 30*time.Minute)
	if err != nil || len(trafficMetrics) < 5 {
		return true, 0
	}

	mid := len(trafficMetrics) / 2
	earlyAvg := CalculateAverage(trafficMetrics[:mid])
	lateAvg := CalculateAverage(trafficMetrics[mid:])

	if earlyAvg == 0 {
		earlyAvg = 1.0
	}

	growthPercent = ((lateAvg - earlyAvg) / earlyAvg) * 100
	stable = math.Abs(growthPercent) < 20.0

	return stable, growthPercent
}

func (m *MemoryLeakDetector) detectAcceleration(metrics []*storage.Metric) bool {
	if len(metrics) < 6 {
		return false
	}

	third := len(metrics) / 3
	seg1 := metrics[:third]
	seg2 := metrics[third : 2*third]
	seg3 := metrics[2*third:]

	slope1, _, _, _ := PerformLinearRegression(seg1)
	slope2, _, _, _ := PerformLinearRegression(seg2)
	slope3, _, _, _ := PerformLinearRegression(seg3)

	return slope3 > slope2 && slope2 > slope1 && slope1 > 0
}

func (m *MemoryLeakDetector) verifySustainedGrowth(metrics []*storage.Metric) bool {
	if len(metrics) < 5 {
		return false
	}

	growthCount := 0
	for i := 2; i < len(metrics); i++ {
		movingAvg := CalculateAverage(metrics[i-2 : i])
		if metrics[i].MetricValue > movingAvg {
			growthCount++
		}
	}

	return float64(growthCount)/float64(len(metrics)-2) > 0.7
}

func (m *MemoryLeakDetector) getMemoryMetrics(ctx context.Context, serviceName string, duration time.Duration) ([]*storage.Metric, error) {
	metricNames := []string{
		"memory_usage",
		"memory_usage_percent",
		"memory_usage_mb",
		"memory_working_set_bytes",
	}

	for _, name := range metricNames {
		metrics, err := m.db.GetRecentMetrics(ctx, serviceName, name, duration)
		if err == nil && len(metrics) > 0 {
			return metrics, nil
		}
	}

	return nil, fmt.Errorf("no memory metrics found")
} //ek duration ke baad ke saare metrics mil jaenge 

func (m *MemoryLeakDetector) buildRecommendation(detected bool, severity string, growthRate, currentMemory float64, trafficStable, accelerating bool) string {
	if !detected {
		return "No memory leak detected. Memory usage is stable."
	}

	recommendation := ""
	switch severity {
	case "CRITICAL":
		recommendation = "CRITICAL MEMORY LEAK: Immediate action required. "
	case "HIGH":
		recommendation = "HIGH PRIORITY: Memory leak detected. "
	default:
		recommendation = "MEMORY LEAK WARNING: "
	}

	recommendation += fmt.Sprintf("Memory growing at %.2f%% per minute. ", growthRate)

	if accelerating {
		recommendation += "Growth is ACCELERATING - this is urgent. "
	}

	if currentMemory > 80.0 {
		recommendation += "Current memory usage critical (>80%%). "
	}

	// Add traffic pattern context
	if !trafficStable {
		recommendation += "Note: Traffic is unstable, which may explain some memory variation. "
	} else {
		recommendation += "Memory growth is occurring despite stable traffic. "
	}

	recommendation += "Actions: 1) Enable heap profiling. 2) Check for unclosed connections/resources. 3) Review recent code changes for memory allocations. "

	if severity == "CRITICAL" {
		recommendation += "4) Consider immediate restart to prevent OOM. "
	}

	return recommendation
}

func (m *MemoryLeakDetector) calculateSeverity(confidence, growthRate, currentMemory float64) string {
	if confidence < 80 {
		return "LOW"
	}
	if growthRate > 2.0 || currentMemory > 85.0 {
		return "CRITICAL"
	}
	if growthRate > 1.0 || currentMemory > 75.0 {
		return "HIGH"
	}
	if growthRate > 0.5 || currentMemory > 65.0 {
		return "MEDIUM"
	}
	return "LOW"
}
