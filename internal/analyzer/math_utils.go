package analyzer

import (
	"math"

	"github.com/namansh70747/AURA-Autonomous-Unified-Reliability-Automation-Platform/internal/storage"
)

func PerformLinearRegression(metrics []*storage.Metric) (slope, intercept, rSquared, growthRatePercent float64) {
	if len(metrics) < 2 {
		return 0, 0, 0, 0
	}

	n := float64(len(metrics))
	var sumX, sumY, sumXY, sumX2 float64
	startTime := metrics[0].Timestamp.Unix()

	for _, metric := range metrics {
		x := float64(metric.Timestamp.Unix()-startTime) / 60.0 // Convert to minutes
		y := metric.MetricValue
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	numerator := n*sumXY - sumX*sumY
	denominator := n*sumX2 - sumX*sumX

	if denominator == 0 {
		return 0, 0, 0, 0
	}

	slope = numerator / denominator
	intercept = (sumY - slope*sumX) / n
	meanY := sumY / n

	var ssTotal, ssResidual float64
	for _, metric := range metrics {
		x := float64(metric.Timestamp.Unix()-startTime) / 60.0
		y := metric.MetricValue
		predicted := slope*x + intercept
		ssTotal += math.Pow(y-meanY, 2)
		ssResidual += math.Pow(y-predicted, 2)
	}

	if ssTotal == 0 {
		rSquared = 1.0
	} else {
		rSquared = 1 - (ssResidual / ssTotal)
		if rSquared < 0 {
			rSquared = 0
		}
	}

	if meanY > 0 {
		growthRatePercent = (slope / meanY) * 100
	}

	return slope, intercept, rSquared, growthRatePercent 
	// how fast memory is increasing / decreasing
	// starting point (not very used here, but needed to complete linear regression formula)
	// how straight / how consistent the line is (0 to 1)
	// how fast it is growing in percentage
	/*
	| r² value | meaning in simple english                                       |
	| -------- | --------------------------------------------------------------- |
	| 1.0      | perfectly consistent trend. Every point is exactly on the line. |
	| 0.9      | very consistent. Almost a clean line.                           |
	| 0.7      | medium ok. Some noise.                                          |
	| 0.3      | very noisy. Points scattered.                                   |
	| 0.0      | no trend. pure random.                                          |
	*/
}

func PerformLinearRegressionOnValues(x, y []float64) (slope, intercept, rSquared float64) {
	n := float64(len(x))
	if n == 0 || len(x) != len(y) {
		return 0, 0, 0
	}

	var sumX, sumY, sumXY, sumX2 float64
	for i := range x {
		sumX += x[i]
		sumY += y[i]
		sumXY += x[i] * y[i]
		sumX2 += x[i] * x[i]
	}

	meanX := sumX / n
	meanY := sumY / n

	numerator := sumXY - n*meanX*meanY
	denominator := sumX2 - n*meanX*meanX

	if denominator == 0 {
		return 0, meanY, 0
	}

	slope = numerator / denominator
	intercept = meanY - slope*meanX

	var ssRes, ssTot float64
	for i := range x {
		predicted := slope*x[i] + intercept
		ssRes += (y[i] - predicted) * (y[i] - predicted)
		ssTot += (y[i] - meanY) * (y[i] - meanY)
	}

	if ssTot == 0 {
		rSquared = 1.0
	} else {
		rSquared = 1 - (ssRes / ssTot)
	}

	return slope, intercept, rSquared
}

func CalculateAverage(metrics []*storage.Metric) float64 {
	if len(metrics) == 0 {
		return 0
	}
	var sum float64
	for _, m := range metrics {
		sum += m.MetricValue
	}
	return sum / float64(len(metrics))
}

// CalculateAverageFromRecords computes average from metric records
func CalculateAverageFromRecords(records []storage.MetricRecord) float64 {
	if len(records) == 0 {
		return 0
	}
	var sum float64
	for _, r := range records {
		sum += r.Value
	}
	return sum / float64(len(records))
}

// CalculateAverageFromValues computes average from float slice
func CalculateAverageFromValues(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// CalculateMax finds maximum value in metrics
func CalculateMax(metrics []*storage.Metric) float64 {
	if len(metrics) == 0 {
		return 0
	}
	max := metrics[0].MetricValue
	for _, m := range metrics {
		if m.MetricValue > max {
			max = m.MetricValue
		}
	}
	return max
}

// CalculateMin finds minimum value in metrics
func CalculateMin(metrics []*storage.Metric) float64 {
	if len(metrics) == 0 {
		return 0
	}
	min := metrics[0].MetricValue
	for _, m := range metrics {
		if m.MetricValue < min {
			min = m.MetricValue
		}
	}
	return min
}

// CalculateVolatility computes coefficient of variation
func CalculateVolatility(metrics []*storage.Metric) float64 {
	if len(metrics) < 2 {
		return 0
	}

	avg := CalculateAverage(metrics)
	if avg == 0 {
		return 0
	}

	var variance float64
	for _, m := range metrics {
		diff := m.MetricValue - avg
		variance += diff * diff
	}

	stdDev := math.Sqrt(variance / float64(len(metrics)))
	return (stdDev / avg) * 100
}

// CalculateVolatilityFromValues computes coefficient of variation from values
func CalculateVolatilityFromValues(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	mean := CalculateAverageFromValues(values)
	if mean == 0 {
		return 0
	}

	var variance float64
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	stdDev := math.Sqrt(variance / float64(len(values)))

	return stdDev / mean
}

// CalculateStdDev computes standard deviation
func CalculateStdDev(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}

	mean := CalculateAverageFromValues(values)
	var variance float64
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	return math.Sqrt(variance / float64(len(values)))
}

// CalculatePearsonCorrelation calculates Pearson correlation between two metric sets
func CalculatePearsonCorrelation(m1, m2 []*storage.Metric) float64 {
	n := int(math.Min(float64(len(m1)), float64(len(m2))))
	if n < 3 {
		return 0
	}

	var sumX, sumY, sumXY, sumX2, sumY2 float64
	for i := 0; i < n; i++ {
		x := m1[i].MetricValue
		y := m2[i].MetricValue
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
		sumY2 += y * y
	}

	nf := float64(n)
	numerator := nf*sumXY - sumX*sumY
	denominator := math.Sqrt((nf*sumX2 - sumX*sumX) * (nf*sumY2 - sumY*sumY))

	if denominator == 0 {
		return 0
	}

	return math.Abs(numerator / denominator)
}

// RecordsToValues converts metric records to float slice
func RecordsToValues(records []storage.MetricRecord) []float64 {
	values := make([]float64, len(records))
	for i, r := range records {
		values[i] = r.Value
	}
	return values
}

// HasAnomalies detects statistical anomalies using Z-score
func HasAnomalies(values []float64, threshold float64) bool {
	if len(values) < 3 {
		return false
	}

	mean := CalculateAverageFromValues(values)
	stdDev := CalculateStdDev(values)

	if stdDev == 0 {
		return false
	}

	for _, v := range values {
		zScore := math.Abs(v-mean) / stdDev
		if zScore > threshold {
			return true
		}
	}

	return false
}

// CalculatePercentile calculates the nth percentile
func CalculatePercentile(values []float64, percentile float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sorted := make([]float64, len(values))
	copy(sorted, values)

	// Simple bubble sort for small datasets
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
