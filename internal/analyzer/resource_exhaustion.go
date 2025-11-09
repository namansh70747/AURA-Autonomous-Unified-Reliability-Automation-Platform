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

// Analyze detects resource exhaustion using multi-dimensional analysis
func (r *ResourceExhaustionDetector) Analyze(ctx context.Context, serviceName string) (*Detection, error) {
	logger.Info("Starting resource exhaustion analysis", zap.String("service", serviceName))

	confidence := 0.0 // Confidence score = 0.0
	evidence := make(map[string]interface{})// evidence ka map[string]interface{}

	// 1. CPU exhaustion analysis
	cpuExhausted, cpuUsage, cpuTrend := r.analyzeCPUExhaustion(ctx, serviceName)
	if cpuExhausted {
		confidence += 40.0
		evidence["cpu_exhausted"] = true
		evidence["cpu_usage_percent"] = fmt.Sprintf("%.1f", cpuUsage)
		evidence["cpu_trend"] = cpuTrend
	}

	// 2. Memory exhaustion analysis
	memExhausted, memUsage, memTrend := r.analyzeMemoryExhaustion(ctx, serviceName)
	if memExhausted {
		confidence += 40.0
		evidence["memory_exhausted"] = true
		evidence["memory_usage_percent"] = fmt.Sprintf("%.1f", memUsage)
		evidence["memory_trend"] = memTrend
	}

	// 3. Predictive time-to-exhaustion
	if cpuTrend == "increasing" || memTrend == "increasing" {
		eta := r.predictExhaustionTime(ctx, serviceName, cpuTrend == "increasing", memTrend == "increasing")
		if eta > 0 && eta < 60 {
			confidence += 20.0
			evidence["exhaustion_eta_min"] = fmt.Sprintf("%.0f", eta)
			evidence["critical_window"] = true
		}
	}

	// 4. Traffic correlation check
	trafficHigh := r.isTrafficHigh(ctx, serviceName)
	if trafficHigh {
		evidence["high_traffic_detected"] = true
		evidence["note"] = "Resource exhaustion may be load-related"
	} else if cpuExhausted || memExhausted {
		confidence += 10.0
		evidence["exhaustion_under_normal_load"] = true
	}

	detected := confidence > 70.0
	severity := r.calculateSeverity(confidence, cpuUsage, memUsage)
	recommendation := r.buildRecommendation(detected, severity, cpuExhausted, memExhausted, trafficHigh)

	return &Detection{
		Type:           DetectionResourceExhaustion,
		ServiceName:    serviceName,
		Detected:       detected,
		Confidence:     confidence,
		Timestamp:      time.Now(),
		Evidence:       evidence,
		Recommendation: recommendation,
		Severity:       severity,
	}, nil
}

func (r *ResourceExhaustionDetector) analyzeCPUExhaustion(ctx context.Context, serviceName string) (exhausted bool, usage float64, trend string) {
	cpuMetrics, err := r.db.GetRecentMetrics(ctx, serviceName, "cpu_usage", 10*time.Minute)
	if err != nil || len(cpuMetrics) < 3 {
		return false, 0, "unknown"
	} //error waali state 

	usage = cpuMetrics[len(cpuMetrics)-1].MetricValue //usage of last metric 
	avgUsage := CalculateAverage(cpuMetrics) // average usage 

	exhausted = avgUsage > 85.0 && usage > 80.0 // avg usage and usage is greater 
	
	if len(cpuMetrics) > 5 {
		mid := len(cpuMetrics) / 2
		first := CalculateAverage(cpuMetrics[:mid])
		second := CalculateAverage(cpuMetrics[mid:])
		if second > first+10.0 {
			trend = "increasing"
		} else if second < first-10.0 {
			trend = "decreasing"
		} else {
			trend = "stable"
		}
	}

	return exhausted, usage, trend
}

func (r *ResourceExhaustionDetector) analyzeMemoryExhaustion(ctx context.Context, serviceName string) (exhausted bool, usage float64, trend string) {
	memMetrics, err := r.db.GetRecentMetrics(ctx, serviceName, "memory_usage", 10*time.Minute)
	if err != nil || len(memMetrics) < 3 {
		return false, 0, "unknown"
	}

	usage = memMetrics[len(memMetrics)-1].MetricValue
	avgUsage := CalculateAverage(memMetrics)

	exhausted = avgUsage > 85.0 && usage > 80.0

	if len(memMetrics) > 5 {
		mid := len(memMetrics) / 2
		first := CalculateAverage(memMetrics[:mid])
		second := CalculateAverage(memMetrics[mid:])
		if second > first+10.0 {
			trend = "increasing"
		} else if second < first-10.0 {
			trend = "decreasing"
		} else {
			trend = "stable"
		}
	}

	return exhausted, usage, trend
}

func (r *ResourceExhaustionDetector) predictExhaustionTime(ctx context.Context, serviceName string, cpuIncreasing, memIncreasing bool) float64 {
	if cpuIncreasing {
		cpuMetrics, err := r.db.GetRecentMetrics(ctx, serviceName, "cpu_usage", 15*time.Minute)
		if err == nil && len(cpuMetrics) > 3 {
			slope, _, _, _ := PerformLinearRegression(cpuMetrics)
			if slope > 0 {
				current := cpuMetrics[len(cpuMetrics)-1].MetricValue
				remaining := 100.0 - current
				return remaining / (slope * 60) // minutes to 100%
			}
		}
	}

	if memIncreasing {
		memMetrics, err := r.db.GetRecentMetrics(ctx, serviceName, "memory_usage", 15*time.Minute)
		if err == nil && len(memMetrics) > 3 {
			slope, _, _, _ := PerformLinearRegression(memMetrics)
			if slope > 0 {
				current := memMetrics[len(memMetrics)-1].MetricValue
				remaining := 100.0 - current
				return remaining / (slope * 60)
			}
		}
	}

	return -1
}

func (r *ResourceExhaustionDetector) isTrafficHigh(ctx context.Context, serviceName string) bool {
	metrics, err := r.db.GetRecentMetrics(ctx, serviceName, "request_rate", 10*time.Minute)
	if err != nil || len(metrics) < 3 {
		return false
	}

	avg := CalculateAverage(metrics)
	return avg > 100.0 // Threshold for "high" traffic
}

func (r *ResourceExhaustionDetector) buildRecommendation(detected bool, severity string, cpuExhausted, memExhausted, trafficHigh bool) string {
	if !detected {
		return "No resource exhaustion detected. Resource usage is within normal limits."
	}

	rec := ""
	if severity == "CRITICAL" {
		rec = "CRITICAL RESOURCE EXHAUSTION: Immediate action required. "
	} else {
		rec = "RESOURCE EXHAUSTION WARNING: "
	}

	if cpuExhausted && memExhausted {
		rec += "Both CPU and memory are exhausted. "
	} else if cpuExhausted {
		rec += "CPU is exhausted. "
	} else if memExhausted {
		rec += "Memory is exhausted. "
	}

	if trafficHigh {
		rec += "High traffic detected - consider horizontal scaling. "
	} else {
		rec += "Exhaustion under normal load - investigate efficiency issues. "
	}

	rec += "Actions: 1) Scale up resources immediately. 2) Check for resource leaks. 3) Review recent deployments. 4) Enable resource profiling."

	return rec
}

func (r *ResourceExhaustionDetector) calculateSeverity(confidence, cpuUsage, memUsage float64) string {
	if confidence < 70 {
		return "LOW"
	}
	if cpuUsage > 95.0 || memUsage > 95.0 {
		return "CRITICAL"
	}
	if cpuUsage > 90.0 || memUsage > 90.0 {
		return "HIGH"
	}
	return "MEDIUM"
}
