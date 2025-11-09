package analyzer

import (
	"math"
	"time"

	"github.com/namansh70747/AURA-Autonomous-Unified-Reliability-Automation-Platform/internal/storage"
)

// PatternMatcher detects patterns and trends in metric time series
type PatternMatcher struct {
	db *storage.PostgresClient
}

// NewPatternMatcher creates a new pattern matcher
func NewPatternMatcher(db *storage.PostgresClient) *PatternMatcher {
	return &PatternMatcher{db: db}
}

type TrendResult struct {
	Direction   string
	Slope       float64
	RSquared    float64
	Volatility  float64
	ChangePoint *time.Time
}

func (pm *PatternMatcher) DetectTrend(serviceName, metricName string, duration time.Duration) (*TrendResult, error) {
	endTime := time.Now()
	startTime := endTime.Add(-duration)

	metrics, err := pm.db.GetMetricsInRange(serviceName, metricName, startTime, endTime)
	if err != nil {
		return nil, err
	}

	if len(metrics) < 5 {
		return &TrendResult{Direction: "insufficient_data"}, nil
	}

	// Prepare data for regression
	var xValues, yValues []float64
	baseTime := metrics[0].Timestamp.Unix()

	for _, m := range metrics {
		x := float64(m.Timestamp.Unix() - baseTime)
		xValues = append(xValues, x)
		yValues = append(yValues, m.Value)
	}

	slope, _, rSquared := PerformLinearRegressionOnValues(xValues, yValues)
	volatility := CalculateVolatilityFromValues(yValues)
	changePoint := pm.detectChangePoints(metrics)

	direction := "stable"
	if math.Abs(slope) > 0.001 {
		if slope > 0 {
			direction = "increasing"
		} else {
			direction = "decreasing"
		}
	}

	return &TrendResult{
		Direction:   direction,
		Slope:       slope,
		RSquared:    rSquared,
		Volatility:  volatility,
		ChangePoint: changePoint,
	}, nil
}

// detectChangePoints identifies significant change points
func (pm *PatternMatcher) detectChangePoints(metrics []storage.MetricRecord) *time.Time {
	if len(metrics) < 10 {
		return nil
	}

	mid := len(metrics) / 2

	var sum1, sum2 float64
	for i := 0; i < mid; i++ {
		sum1 += metrics[i].Value
	}
	for i := mid; i < len(metrics); i++ {
		sum2 += metrics[i].Value
	}

	mean1 := sum1 / float64(mid)
	mean2 := sum2 / float64(len(metrics)-mid)

	var variance1, variance2 float64
	for i := 0; i < mid; i++ {
		diff := metrics[i].Value - mean1
		variance1 += diff * diff
	}
	for i := mid; i < len(metrics); i++ {
		diff := metrics[i].Value - mean2
		variance2 += diff * diff
	}

	pooledStdDev := math.Sqrt((variance1 + variance2) / float64(len(metrics)))

	if math.Abs(mean2-mean1) > 2*pooledStdDev {
		changeTime := metrics[mid].Timestamp
		return &changeTime
	}

	return nil
}

// CompareMetricBehavior compares current vs baseline behavior
func (pm *PatternMatcher) CompareMetricBehavior(serviceName, metricName string, currentDuration, baselineDuration time.Duration) (float64, error) {
	endTime := time.Now()
	currentStart := endTime.Add(-currentDuration)
	baselineStart := endTime.Add(-baselineDuration)
	baselineEnd := endTime.Add(-currentDuration)

	currentMetrics, err := pm.db.GetMetricsInRange(serviceName, metricName, currentStart, endTime)
	if err != nil {
		return 0, err
	}

	baselineMetrics, err := pm.db.GetMetricsInRange(serviceName, metricName, baselineStart, baselineEnd)
	if err != nil {
		return 0, err
	}

	if len(currentMetrics) < 3 || len(baselineMetrics) < 3 {
		return 0, nil
	}

	currentMean := CalculateAverageFromRecords(currentMetrics)
	baselineMean := CalculateAverageFromRecords(baselineMetrics)

	if baselineMean == 0 {
		return 0, nil
	}

	return ((currentMean - baselineMean) / baselineMean) * 100, nil
}

// DetectSeasonality checks for periodic patterns
func (pm *PatternMatcher) DetectSeasonality(serviceName, metricName string, duration time.Duration) (bool, time.Duration, error) {
	endTime := time.Now()
	startTime := endTime.Add(-duration)

	metrics, err := pm.db.GetMetricsInRange(serviceName, metricName, startTime, endTime)
	if err != nil {
		return false, 0, err
	}

	if len(metrics) < 20 {
		return false, 0, nil
	}

	periods := []time.Duration{
		1 * time.Hour,
		24 * time.Hour,
	}

	for _, period := range periods {
		autocorr := pm.calculateAutocorrelation(metrics, period)
		if autocorr > 0.7 {
			return true, period, nil
		}
	}

	return false, 0, nil
}

// calculateAutocorrelation calculates autocorrelation at given lag
func (pm *PatternMatcher) calculateAutocorrelation(metrics []storage.MetricRecord, lag time.Duration) float64 {
	if len(metrics) < 2 {
		return 0
	}

	var sum float64
	for _, m := range metrics {
		sum += m.Value
	}
	mean := sum / float64(len(metrics))

	lagSeconds := int64(lag.Seconds())
	var numerator, denominator float64
	matchCount := 0

	for i := 0; i < len(metrics); i++ {
		targetTime := metrics[i].Timestamp.Add(lag)
		for j := 0; j < len(metrics); j++ {
			if math.Abs(float64(metrics[j].Timestamp.Unix()-targetTime.Unix())) < float64(lagSeconds)/2 {
				numerator += (metrics[i].Value - mean) * (metrics[j].Value - mean)
				denominator += (metrics[i].Value - mean) * (metrics[i].Value - mean)
				matchCount++
				break
			}
		}
	}

	if denominator == 0 || matchCount < 3 {
		return 0
	}

	return numerator / denominator
}
