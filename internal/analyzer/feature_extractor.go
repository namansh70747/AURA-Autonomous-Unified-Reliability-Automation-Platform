package analyzer

import (
	"context"
	"math"
	"time"

	"github.com/namansh70747/AURA-Autonomous-Unified-Reliability-Automation-Platform/internal/storage"
)

// FeatureExtractor extracts 60+ dimensional features from raw metrics
type FeatureExtractor struct {
	db *storage.PostgresClient
}

func NewFeatureExtractor(db *storage.PostgresClient) *FeatureExtractor {
	return &FeatureExtractor{db: db}
}

// ServiceFeatures represents comprehensive feature set
type ServiceFeatures struct {
	ServiceName string
	Timestamp   time.Time

	// Time-domain features (CPU)
	CPUMean            float64
	CPUStdDev          float64
	CPUMin             float64
	CPUMax             float64
	CPURange           float64
	CPUTrend           float64 // slope (% per minute)
	CPUVolatility      float64
	CPUAutocorrelation float64
	CPUAnomalyScore    float64

	// Time-domain features (Memory)
	MemoryMean            float64
	MemoryStdDev          float64
	MemoryMin             float64
	MemoryMax             float64
	MemoryRange           float64
	MemoryTrend           float64
	MemoryVolatility      float64
	MemoryAutocorrelation float64
	MemoryAnomalyScore    float64

	// Error rate features
	ErrorRateMean      float64
	ErrorRateMax       float64
	ErrorRateTrend     float64
	ErrorRateSpikiness float64
	ErrorAnomalyScore  float64

	// Latency features
	LatencyMean         float64
	LatencyP50          float64
	LatencyP95          float64
	LatencyP99          float64
	LatencyStdDev       float64
	LatencyAnomalyScore float64

	// Cross-metric correlations
	CPUMemoryCorr    float64
	CPUErrorCorr     float64
	MemoryErrorCorr  float64
	LatencyErrorCorr float64
	RequestCPUCorr   float64

	// Pattern detection
	HasPeriodicPattern bool
	PeriodLength       time.Duration
	HasSeasonality     bool
	HasTrend           bool
	TrendDirection     string // "increasing", "decreasing", "stable"

	// Composite scores
	SystemStress        float64 // 0-100
	HealthScore         float64 // 0-100
	StabilityIndex      float64 // 0-10
	PredictabilityScore float64 // 0-100
}

// ExtractFeatures performs comprehensive feature extraction
func (fe *FeatureExtractor) ExtractFeatures(ctx context.Context, serviceName string, window time.Duration) (*ServiceFeatures, error) {
	features := &ServiceFeatures{
		ServiceName: serviceName,
		Timestamp:   time.Now(),
	}

	// Extract CPU features
	cpuMetrics, err := fe.db.GetRecentMetrics(ctx, serviceName, "cpu_usage", window)
	if err == nil && len(cpuMetrics) > 0 {
		fe.extractCPUFeatures(cpuMetrics, features)
	}

	// Try alternative CPU metric names
	if len(cpuMetrics) == 0 {
		cpuMetrics, _ = fe.db.GetRecentMetrics(ctx, serviceName, "cpu_usage_percent", window)
		if len(cpuMetrics) > 0 {
			fe.extractCPUFeatures(cpuMetrics, features)
		}
	}

	// Extract Memory features
	memMetrics, err := fe.db.GetRecentMetrics(ctx, serviceName, "memory_usage", window)
	if err == nil && len(memMetrics) > 0 {
		fe.extractMemoryFeatures(memMetrics, features)
	}

	// Try alternative memory metric names
	if len(memMetrics) == 0 {
		memMetrics, _ = fe.db.GetRecentMetrics(ctx, serviceName, "memory_usage_percent", window)
		if len(memMetrics) > 0 {
			fe.extractMemoryFeatures(memMetrics, features)
		}
	}

	// Extract Error features
	errorMetrics, err := fe.db.GetRecentMetrics(ctx, serviceName, "error_rate", window)
	if err == nil && len(errorMetrics) > 0 {
		fe.extractErrorFeatures(errorMetrics, features)
	}

	// Try alternative error metric names
	if len(errorMetrics) == 0 {
		errorMetrics, _ = fe.db.GetRecentMetrics(ctx, serviceName, "app_errors_total", window)
		if len(errorMetrics) > 0 {
			fe.extractErrorFeatures(errorMetrics, features)
		}
	}
	if len(errorMetrics) == 0 {
		errorMetrics, _ = fe.db.GetRecentMetrics(ctx, serviceName, "error_count", window)
		if len(errorMetrics) > 0 {
			fe.extractErrorFeatures(errorMetrics, features)
		}
	} // Extract Latency features
	latencyMetrics, err := fe.db.GetRecentMetrics(ctx, serviceName, "response_time", window)
	if err == nil && len(latencyMetrics) > 0 {
		fe.extractLatencyFeatures(latencyMetrics, features)
	}

	// Try alternative latency metric names
	if len(latencyMetrics) == 0 {
		latencyMetrics, _ = fe.db.GetRecentMetrics(ctx, serviceName, "response_time_p95_ms", window)
		if len(latencyMetrics) > 0 {
			fe.extractLatencyFeatures(latencyMetrics, features)
		}
	}

	// Calculate cross-metric correlations
	if len(cpuMetrics) > 0 && len(memMetrics) > 0 {
		features.CPUMemoryCorr = CalculatePearsonCorrelation(cpuMetrics, memMetrics)
	}

	if len(cpuMetrics) > 0 && len(errorMetrics) > 0 {
		features.CPUErrorCorr = CalculatePearsonCorrelation(cpuMetrics, errorMetrics)
	}

	if len(memMetrics) > 0 && len(errorMetrics) > 0 {
		features.MemoryErrorCorr = CalculatePearsonCorrelation(memMetrics, errorMetrics)
	}

	if len(latencyMetrics) > 0 && len(errorMetrics) > 0 {
		features.LatencyErrorCorr = CalculatePearsonCorrelation(latencyMetrics, errorMetrics)
	}

	// Pattern detection
	if len(cpuMetrics) > 10 {
		fe.detectPatterns(cpuMetrics, features)
	}

	// Calculate composite scores
	fe.calculateCompositeScores(features)

	return features, nil
}

func (fe *FeatureExtractor) extractCPUFeatures(metrics []*storage.Metric, features *ServiceFeatures) {
	values := extractMetricValues(metrics)

	features.CPUMean = CalculateMean(values)
	features.CPUStdDev = CalculateStdDev(values)
	features.CPUMin = minFloat64(values)
	features.CPUMax = maxFloat64(values)
	features.CPURange = features.CPUMax - features.CPUMin

	slope, _, _, _ := PerformLinearRegression(metrics)
	features.CPUTrend = slope

	if features.CPUMean > 0 {
		features.CPUVolatility = features.CPUStdDev / features.CPUMean
	}
	features.CPUAutocorrelation = calculateAutocorrelation(values, 1)
	features.CPUAnomalyScore = calculateAnomalyScore(values)
}

func (fe *FeatureExtractor) extractMemoryFeatures(metrics []*storage.Metric, features *ServiceFeatures) {
	values := extractMetricValues(metrics)

	features.MemoryMean = CalculateMean(values)
	features.MemoryStdDev = CalculateStdDev(values)
	features.MemoryMin = minFloat64(values)
	features.MemoryMax = maxFloat64(values)
	features.MemoryRange = features.MemoryMax - features.MemoryMin

	slope, _, _, _ := PerformLinearRegression(metrics)
	features.MemoryTrend = slope

	if features.MemoryMean > 0 {
		features.MemoryVolatility = features.MemoryStdDev / features.MemoryMean
	}
	features.MemoryAutocorrelation = calculateAutocorrelation(values, 1)
	features.MemoryAnomalyScore = calculateAnomalyScore(values)
}

func (fe *FeatureExtractor) extractErrorFeatures(metrics []*storage.Metric, features *ServiceFeatures) {
	values := extractMetricValues(metrics)

	features.ErrorRateMean = CalculateMean(values)
	features.ErrorRateMax = maxFloat64(values)

	slope, _, _, _ := PerformLinearRegression(metrics)
	features.ErrorRateTrend = slope

	features.ErrorRateSpikiness = calculateSpikiness(values)
	features.ErrorAnomalyScore = calculateAnomalyScore(values)
}

func (fe *FeatureExtractor) extractLatencyFeatures(metrics []*storage.Metric, features *ServiceFeatures) {
	values := extractMetricValues(metrics)

	features.LatencyMean = CalculateMean(values)
	features.LatencyP50 = CalculatePercentile(values, 50)
	features.LatencyP95 = CalculatePercentile(values, 95)
	features.LatencyP99 = CalculatePercentile(values, 99)
	features.LatencyStdDev = CalculateStdDev(values)
	features.LatencyAnomalyScore = calculateAnomalyScore(values)
}

func (fe *FeatureExtractor) detectPatterns(metrics []*storage.Metric, features *ServiceFeatures) {
	values := extractMetricValues(metrics)

	// Detect periodicity using autocorrelation
	maxLag := len(values) / 3
	if maxLag > 20 {
		maxLag = 20
	}

	if maxLag < 2 {
		return
	}

	autocorrs := make([]float64, maxLag)
	for lag := 1; lag < maxLag; lag++ {
		autocorrs[lag] = calculateAutocorrelation(values, lag)
	}

	// Find peak autocorrelation (excluding lag 0)
	maxAutocorr := 0.0
	maxLagIdx := 0
	for i := 1; i < len(autocorrs); i++ {
		if autocorrs[i] > maxAutocorr {
			maxAutocorr = autocorrs[i]
			maxLagIdx = i
		}
	}

	if maxAutocorr > 0.5 { // Strong periodicity
		features.HasPeriodicPattern = true
		features.PeriodLength = time.Duration(maxLagIdx*5) * time.Second
	}

	// Detect trend
	slope, _, _, _ := PerformLinearRegression(metrics)
	if math.Abs(slope) > 0.1 {
		features.HasTrend = true
		if slope > 0 {
			features.TrendDirection = "increasing"
		} else {
			features.TrendDirection = "decreasing"
		}
	} else {
		features.TrendDirection = "stable"
	}
}

func (fe *FeatureExtractor) calculateCompositeScores(features *ServiceFeatures) {
	// System Stress (0-100): combination of CPU, Memory, Errors
	cpuStress := features.CPUMean
	memStress := features.MemoryMean
	errorStress := features.ErrorRateMean * 10 // scale up error rate
	features.SystemStress = (cpuStress + memStress + errorStress) / 3
	if features.SystemStress > 100 {
		features.SystemStress = 100
	}

	// Health Score (0-100): inverse of problems
	healthDeductions := 0.0
	if features.CPUMean > 80 {
		healthDeductions += 20
	}
	if features.MemoryMean > 85 {
		healthDeductions += 20
	}
	if features.ErrorRateMean > 5 {
		healthDeductions += 30
	}
	if features.LatencyP95 > 2000 {
		healthDeductions += 15
	}
	if features.CPUTrend > 0.5 {
		healthDeductions += 10 // growing CPU
	}
	if features.MemoryTrend > 0.5 {
		healthDeductions += 10 // growing memory (leak?)
	}
	features.HealthScore = math.Max(0, 100-healthDeductions)

	// Stability Index (0-10): lower volatility = higher stability
	cpuStability := 10 * (1 - math.Min(1, features.CPUVolatility))
	memStability := 10 * (1 - math.Min(1, features.MemoryVolatility))
	features.StabilityIndex = (cpuStability + memStability) / 2

	// Predictability Score (0-100): based on patterns and autocorrelation
	predictability := 50.0
	if features.HasPeriodicPattern {
		predictability += 20
	}
	if features.CPUAutocorrelation > 0.7 {
		predictability += 15
	}
	if features.TrendDirection != "stable" {
		predictability += 10 // trends are predictable
	}
	features.PredictabilityScore = math.Min(100, predictability)
}

// ==================== HELPER FUNCTIONS ====================

func extractMetricValues(metrics []*storage.Metric) []float64 {
	values := make([]float64, len(metrics))
	for i, m := range metrics {
		values[i] = m.MetricValue
	}
	return values
}

func calculateAutocorrelation(values []float64, lag int) float64 {
	if lag >= len(values) || len(values) < 2 {
		return 0
	}

	mean := CalculateMean(values)
	variance := 0.0

	for _, v := range values {
		variance += (v - mean) * (v - mean)
	}

	if variance == 0 {
		return 0
	}

	covariance := 0.0
	for i := 0; i < len(values)-lag; i++ {
		covariance += (values[i] - mean) * (values[i+lag] - mean)
	}

	return covariance / variance
}

func calculateAnomalyScore(values []float64) float64 {
	if len(values) < 3 {
		return 0
	}

	mean := CalculateMean(values)
	stdDev := CalculateStdDev(values)

	if stdDev == 0 {
		return 0
	}

	anomalyCount := 0
	for _, v := range values {
		zScore := math.Abs((v - mean) / stdDev)
		if zScore > 2.5 { // 2.5 sigma threshold
			anomalyCount++
		}
	}

	return (float64(anomalyCount) / float64(len(values))) * 100
}

func calculateSpikiness(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}

	// Calculate successive differences
	diffs := make([]float64, len(values)-1)
	for i := 1; i < len(values); i++ {
		diffs[i-1] = math.Abs(values[i] - values[i-1])
	}

	meanDiff := CalculateMean(diffs)
	stdDevDiff := CalculateStdDev(diffs)

	if meanDiff == 0 {
		return 0
	}

	return stdDevDiff / meanDiff // Coefficient of variation
}

func minFloat64(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	min := values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

func maxFloat64(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	max := values[0]
	for _, v := range values[1:] {
		if v > max {
			max = v
		}
	}
	return max
}
