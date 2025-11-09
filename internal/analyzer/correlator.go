package analyzer

import (
	"math"
	"time"

	"github.com/namansh70747/AURA-Autonomous-Unified-Reliability-Automation-Platform/internal/storage"
)

// ServiceCorrelator analyzes correlations between services and metrics
type ServiceCorrelator struct {
	db *storage.PostgresClient
}

// NewServiceCorrelator creates a new service correlator
func NewServiceCorrelator(db *storage.PostgresClient) *ServiceCorrelator {
	return &ServiceCorrelator{db: db}
}

// CorrelationResult contains correlation analysis results
type CorrelationResult struct {
	Service1    string
	Service2    string
	Metric1     string
	Metric2     string
	Correlation float64 // Pearson correlation (-1 to 1)
	Lag         time.Duration
	Strength    string // "strong", "moderate", "weak", "none"
	CascadeRisk float64
}

// CalculatePearsonCorrelation calculates Pearson correlation between two metrics
func (sc *ServiceCorrelator) CalculatePearsonCorrelation(service1, metric1, service2, metric2 string, duration time.Duration) (*CorrelationResult, error) {
	endTime := time.Now()
	startTime := endTime.Add(-duration)

	metrics1, err := sc.db.GetMetricsInRange(service1, metric1, startTime, endTime)
	if err != nil {
		return nil, err
	}

	metrics2, err := sc.db.GetMetricsInRange(service2, metric2, startTime, endTime)
	if err != nil {
		return nil, err
	}

	if len(metrics1) < 3 || len(metrics2) < 3 {
		return &CorrelationResult{
			Service1: service1,
			Service2: service2,
			Metric1:  metric1,
			Metric2:  metric2,
			Strength: "insufficient_data",
		}, nil
	}

	// Align metrics by timestamp (find matching pairs)
	var values1, values2 []float64
	for _, m1 := range metrics1 {
		for _, m2 := range metrics2 {
			if math.Abs(float64(m1.Timestamp.Unix()-m2.Timestamp.Unix())) < 30 { // 30 sec tolerance
				values1 = append(values1, m1.Value)
				values2 = append(values2, m2.Value)
				break
			}
		}
	}

	if len(values1) < 3 {
		return &CorrelationResult{
			Service1: service1,
			Service2: service2,
			Metric1:  metric1,
			Metric2:  metric2,
			Strength: "no_overlap",
		}, nil
	}

	correlation := sc.pearsonCorrelation(values1, values2)
	strength := sc.getCorrelationStrength(correlation)
	cascadeRisk := math.Abs(correlation) * 100

	return &CorrelationResult{
		Service1:    service1,
		Service2:    service2,
		Metric1:     metric1,
		Metric2:     metric2,
		Correlation: correlation,
		Strength:    strength,
		CascadeRisk: cascadeRisk,
	}, nil
}

// CalculateCrossCorrelation finds time-lagged correlations
func (sc *ServiceCorrelator) CalculateCrossCorrelation(service1, metric1, service2, metric2 string, duration time.Duration, maxLag time.Duration) (*CorrelationResult, error) {
	endTime := time.Now()
	startTime := endTime.Add(-duration)

	metrics1, err := sc.db.GetMetricsInRange(service1, metric1, startTime, endTime)
	if err != nil {
		return nil, err
	}

	metrics2, err := sc.db.GetMetricsInRange(service2, metric2, startTime, endTime)
	if err != nil {
		return nil, err
	}

	if len(metrics1) < 5 || len(metrics2) < 5 {
		return &CorrelationResult{Strength: "insufficient_data"}, nil
	}

	// Try different lags
	bestCorrelation := 0.0
	bestLag := time.Duration(0)

	lagSteps := []time.Duration{0, 30 * time.Second, 1 * time.Minute, 2 * time.Minute, 5 * time.Minute}
	for _, lag := range lagSteps {
		if lag > maxLag {
			break
		}

		var values1, values2 []float64
		for _, m1 := range metrics1 {
			targetTime := m1.Timestamp.Add(lag)
			for _, m2 := range metrics2 {
				if math.Abs(float64(m2.Timestamp.Unix()-targetTime.Unix())) < 30 {
					values1 = append(values1, m1.Value)
					values2 = append(values2, m2.Value)
					break
				}
			}
		}

		if len(values1) >= 3 {
			corr := sc.pearsonCorrelation(values1, values2)
			if math.Abs(corr) > math.Abs(bestCorrelation) {
				bestCorrelation = corr
				bestLag = lag
			}
		}
	}

	strength := sc.getCorrelationStrength(bestCorrelation)
	cascadeRisk := math.Abs(bestCorrelation) * 100

	return &CorrelationResult{
		Service1:    service1,
		Service2:    service2,
		Metric1:     metric1,
		Metric2:     metric2,
		Correlation: bestCorrelation,
		Lag:         bestLag,
		Strength:    strength,
		CascadeRisk: cascadeRisk,
	}, nil
}

// pearsonCorrelation calculates Pearson correlation coefficient
func (sc *ServiceCorrelator) pearsonCorrelation(x, y []float64) float64 {
	if len(x) != len(y) || len(x) < 2 {
		return 0
	}

	n := float64(len(x))

	// Calculate means
	var sumX, sumY float64
	for i := range x {
		sumX += x[i]
		sumY += y[i]
	}
	meanX := sumX / n
	meanY := sumY / n

	// Calculate correlation
	var numerator, denomX, denomY float64
	for i := range x {
		dx := x[i] - meanX
		dy := y[i] - meanY
		numerator += dx * dy
		denomX += dx * dx
		denomY += dy * dy
	}

	if denomX == 0 || denomY == 0 {
		return 0
	}

	return numerator / math.Sqrt(denomX*denomY)
}

// getCorrelationStrength categorizes correlation strength
func (sc *ServiceCorrelator) getCorrelationStrength(correlation float64) string {
	absCorr := math.Abs(correlation)
	if absCorr >= 0.7 {
		return "strong"
	} else if absCorr >= 0.4 {
		return "moderate"
	} else if absCorr >= 0.2 {
		return "weak"
	}
	return "none"
}

// AnalyzeCascadeRisk assesses risk of cascading failures
func (sc *ServiceCorrelator) AnalyzeCascadeRisk(serviceName string, duration time.Duration) (float64, []string, error) {
	// Get list of other services (simplified - in real scenario, query from config/discovery)
	otherServices := []string{"sample-app", "api-gateway", "database", "cache"}

	var highRiskServices []string
	totalRisk := 0.0
	count := 0

	for _, otherService := range otherServices {
		if otherService == serviceName {
			continue
		}

		// Check error rate correlation
		result, err := sc.CalculatePearsonCorrelation(
			serviceName, "error_rate",
			otherService, "error_rate",
			duration,
		)
		if err != nil {
			continue
		}

		if math.Abs(result.Correlation) > 0.6 {
			highRiskServices = append(highRiskServices, otherService)
			totalRisk += result.CascadeRisk
			count++
		}
	}

	avgRisk := 0.0
	if count > 0 {
		avgRisk = totalRisk / float64(count)
	}

	return avgRisk, highRiskServices, nil
}

// FindCorrelatedMetrics finds all metrics correlated with a given metric
func (sc *ServiceCorrelator) FindCorrelatedMetrics(serviceName, metricName string, duration time.Duration, minCorrelation float64) ([]CorrelationResult, error) {
	// Common metrics to check
	metrics := []string{"cpu_usage", "memory_usage", "error_rate", "response_time", "request_rate"}

	var results []CorrelationResult

	for _, otherMetric := range metrics {
		if otherMetric == metricName {
			continue
		}

		result, err := sc.CalculatePearsonCorrelation(
			serviceName, metricName,
			serviceName, otherMetric,
			duration,
		)
		if err != nil {
			continue
		}

		if math.Abs(result.Correlation) >= minCorrelation {
			results = append(results, *result)
		}
	}

	return results, nil
}
