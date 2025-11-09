package analyzer

import (
	"math"
	"time"

	"github.com/namansh70747/AURA-Autonomous-Unified-Reliability-Automation-Platform/internal/storage"
)

// AnomalyDetector provides multiple statistical methods for anomaly detection
type AnomalyDetector struct {
	db *storage.PostgresClient
}

// NewAnomalyDetector creates a new anomaly detector
func NewAnomalyDetector(db *storage.PostgresClient) *AnomalyDetector {
	return &AnomalyDetector{db: db}
}

// AnomalyResult contains anomaly detection results
type AnomalyResult struct {
	IsAnomaly    bool
	Score        float64 // Severity score (0-100)
	Method       string  // Detection method used
	Threshold    float64
	CurrentValue float64
	ExpectedMin  float64
	ExpectedMax  float64
}

// DetectZScore uses Z-score method (statistical outlier detection)
func (ad *AnomalyDetector) DetectZScore(serviceName, metricName string, duration time.Duration, threshold float64) (*AnomalyResult, error) {
	endTime := time.Now()
	startTime := endTime.Add(-duration)

	metrics, err := ad.db.GetMetricsInRange(serviceName, metricName, startTime, endTime)
	if err != nil {
		return nil, err
	}

	if len(metrics) < 10 {
		return &AnomalyResult{IsAnomaly: false, Method: "zscore", Score: 0}, nil
	}

	var sum float64
	for _, m := range metrics {
		sum += m.Value
	}
	mean := sum / float64(len(metrics))

	var variance float64
	for _, m := range metrics {
		diff := m.Value - mean
		variance += diff * diff
	}
	stdDev := math.Sqrt(variance / float64(len(metrics)))

	latest := metrics[len(metrics)-1].Value
	zScore := math.Abs((latest - mean) / stdDev)

	isAnomaly := zScore > threshold
	score := math.Min((zScore/threshold)*100, 100)

	return &AnomalyResult{
		IsAnomaly:    isAnomaly,
		Score:        score,
		Method:       "zscore",
		Threshold:    threshold,
		CurrentValue: latest,
		ExpectedMin:  mean - threshold*stdDev,
		ExpectedMax:  mean + threshold*stdDev,
	}, nil
}

// DetectIQR uses Interquartile Range method
func (ad *AnomalyDetector) DetectIQR(serviceName, metricName string, duration time.Duration) (*AnomalyResult, error) {
	endTime := time.Now()
	startTime := endTime.Add(-duration)

	metrics, err := ad.db.GetMetricsInRange(serviceName, metricName, startTime, endTime)
	if err != nil {
		return nil, err
	}

	if len(metrics) < 10 {
		return &AnomalyResult{IsAnomaly: false, Method: "iqr", Score: 0}, nil
	}

	values := make([]float64, len(metrics))
	for i, m := range metrics {
		values[i] = m.Value
	}

	q1 := ad.calculatePercentile(values, 25)
	q3 := ad.calculatePercentile(values, 75)
	iqr := q3 - q1

	lowerBound := q1 - 1.5*iqr
	upperBound := q3 + 1.5*iqr

	latest := metrics[len(metrics)-1].Value
	isAnomaly := latest < lowerBound || latest > upperBound

	var score float64
	if isAnomaly {
		if latest < lowerBound {
			score = math.Min(((lowerBound-latest)/iqr)*50, 100)
		} else {
			score = math.Min(((latest-upperBound)/iqr)*50, 100)
		}
	}

	return &AnomalyResult{
		IsAnomaly:    isAnomaly,
		Score:        score,
		Method:       "iqr",
		Threshold:    1.5,
		CurrentValue: latest,
		ExpectedMin:  lowerBound,
		ExpectedMax:  upperBound,
	}, nil
}

// DetectEMA uses Exponential Moving Average method
func (ad *AnomalyDetector) DetectEMA(serviceName, metricName string, duration time.Duration, smoothing float64, threshold float64) (*AnomalyResult, error) {
	endTime := time.Now()
	startTime := endTime.Add(-duration)

	metrics, err := ad.db.GetMetricsInRange(serviceName, metricName, startTime, endTime)
	if err != nil {
		return nil, err
	}

	if len(metrics) < 5 {
		return &AnomalyResult{IsAnomaly: false, Method: "ema", Score: 0}, nil
	}

	alpha := 2.0 / (smoothing + 1.0)
	ema := metrics[0].Value

	for i := 1; i < len(metrics); i++ {
		ema = alpha*metrics[i].Value + (1-alpha)*ema
	}

	latest := metrics[len(metrics)-1].Value
	deviation := math.Abs(latest - ema)

	var sumDeviation float64
	tempEMA := metrics[0].Value
	for i := 1; i < len(metrics); i++ {
		tempEMA = alpha*metrics[i].Value + (1-alpha)*tempEMA
		sumDeviation += math.Pow(metrics[i].Value-tempEMA, 2)
	}
	stdDev := math.Sqrt(sumDeviation / float64(len(metrics)-1))

	isAnomaly := deviation > threshold*stdDev
	score := math.Min((deviation/(threshold*stdDev))*100, 100)

	return &AnomalyResult{
		IsAnomaly:    isAnomaly,
		Score:        score,
		Method:       "ema",
		Threshold:    threshold,
		CurrentValue: latest,
		ExpectedMin:  ema - threshold*stdDev,
		ExpectedMax:  ema + threshold*stdDev,
	}, nil
}

// DetectOscillation detects rapid oscillating behavior
func (ad *AnomalyDetector) DetectOscillation(serviceName, metricName string, duration time.Duration, minChanges int) (*AnomalyResult, error) {
	endTime := time.Now()
	startTime := endTime.Add(-duration)

	metrics, err := ad.db.GetMetricsInRange(serviceName, metricName, startTime, endTime)
	if err != nil {
		return nil, err
	}

	if len(metrics) < 5 {
		return &AnomalyResult{IsAnomaly: false, Method: "oscillation", Score: 0}, nil
	}

	changes := 0
	for i := 2; i < len(metrics); i++ {
		prev := metrics[i-1].Value - metrics[i-2].Value
		curr := metrics[i].Value - metrics[i-1].Value

		if (prev > 0 && curr < 0) || (prev < 0 && curr > 0) {
			changes++
		}
	}

	isAnomaly := changes >= minChanges
	score := math.Min((float64(changes)/float64(minChanges))*100, 100)

	return &AnomalyResult{
		IsAnomaly:    isAnomaly,
		Score:        score,
		Method:       "oscillation",
		Threshold:    float64(minChanges),
		CurrentValue: float64(changes),
	}, nil
}

// DetectCombined uses multiple methods and combines results
func (ad *AnomalyDetector) DetectCombined(serviceName, metricName string, duration time.Duration) (*AnomalyResult, error) {
	zScore, err := ad.DetectZScore(serviceName, metricName, duration, 3.0)
	if err != nil {
		return nil, err
	}

	iqr, err := ad.DetectIQR(serviceName, metricName, duration)
	if err != nil {
		return nil, err
	}

	ema, err := ad.DetectEMA(serviceName, metricName, duration, 10.0, 2.0)
	if err != nil {
		return nil, err
	}

	combinedScore := (zScore.Score*0.4 + iqr.Score*0.3 + ema.Score*0.3)
	isAnomaly := combinedScore > 60

	method := "combined"
	if zScore.IsAnomaly && iqr.IsAnomaly {
		method = "combined(zscore+iqr)"
	} else if zScore.IsAnomaly && ema.IsAnomaly {
		method = "combined(zscore+ema)"
	} else if iqr.IsAnomaly && ema.IsAnomaly {
		method = "combined(iqr+ema)"
	}

	return &AnomalyResult{
		IsAnomaly:    isAnomaly,
		Score:        combinedScore,
		Method:       method,
		Threshold:    60.0,
		CurrentValue: zScore.CurrentValue,
		ExpectedMin:  math.Min(zScore.ExpectedMin, iqr.ExpectedMin),
		ExpectedMax:  math.Max(zScore.ExpectedMax, iqr.ExpectedMax),
	}, nil
}

// calculatePercentile calculates the nth percentile
func (ad *AnomalyDetector) calculatePercentile(values []float64, percentile float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sorted := make([]float64, len(values))
	copy(sorted, values)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	index := (percentile / 100.0) * float64(len(sorted)-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))

	if lower == upper {
		return sorted[lower]
	}

	weight := index - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}
