package analyzer

import (
	"context"
	"fmt"
	"time"

	"github.com/namansh70747/AURA-Autonomous-Unified-Reliability-Automation-Platform/internal/storage"
	"github.com/namansh70747/AURA-Autonomous-Unified-Reliability-Automation-Platform/pkg/logger"
	"go.uber.org/zap"
)

type ExternalFailureDetector struct {
	db *storage.PostgresClient
}

func NewExternalFailureDetector(db *storage.PostgresClient) *ExternalFailureDetector {
	return &ExternalFailureDetector{
		db: db,
	}
}

// Analyze detects external dependency failures
func (e *ExternalFailureDetector) Analyze(ctx context.Context, serviceName string) (*Detection, error) {
	logger.Info("Starting external failure analysis", zap.String("service", serviceName))

	confidence := 0.0
	evidence := make(map[string]interface{})

	// 1. Detect timeout patterns
	timeoutPattern := e.detectTimeoutPattern(ctx, serviceName)
	if timeoutPattern {
		confidence += 35.0
		evidence["timeout_pattern_detected"] = true
	}

	// 2. Detect retry storm
	retryStorm := e.detectRetryStorm(ctx, serviceName)
	if retryStorm {
		confidence += 30.0
		evidence["retry_storm_detected"] = true
	}

	// 3. Check error correlation with external calls
	externalCorrelation := e.analyzeExternalCorrelation(ctx, serviceName)
	if externalCorrelation > 0.7 {
		confidence += 25.0
		evidence["high_external_correlation"] = true
		evidence["correlation_score"] = fmt.Sprintf("%.2f", externalCorrelation)
	}

	// 4. Resource-error mismatch detection
	resourceMismatch := e.detectResourceErrorMismatch(ctx, serviceName)
	if resourceMismatch {
		confidence += 15.0
		evidence["resource_error_mismatch"] = true
		evidence["note"] = "Errors occur without resource spikes - likely external"
	}

	detected := confidence > 70.0
	severity := e.calculateSeverity(confidence, timeoutPattern, retryStorm)
	recommendation := e.buildRecommendation(detected, severity, timeoutPattern, retryStorm)

	return &Detection{
		Type:           DetectionExternalFailure,
		ServiceName:    serviceName,
		Detected:       detected,
		Confidence:     confidence,
		Timestamp:      time.Now(),
		Evidence:       evidence,
		Recommendation: recommendation,
		Severity:       severity,
	}, nil
}

func (e *ExternalFailureDetector) detectTimeoutPattern(ctx context.Context, serviceName string) bool {
	errorMetrics, err := e.db.GetRecentMetrics(ctx, serviceName, "error_rate", 10*time.Minute)
	if err != nil || len(errorMetrics) < 5 {
		return false
	}

	respMetrics, err := e.db.GetRecentMetrics(ctx, serviceName, "response_time", 10*time.Minute)
	if err != nil || len(respMetrics) < 5 {
		return false
	}

	// Check if response time spikes correlate with error spikes
	avgResp := CalculateAverage(respMetrics)
	avgErr := CalculateAverage(errorMetrics)

	// Timeout pattern: high response time + high error rate
	return avgResp > 5000.0 && avgErr > 5.0 // >5s response, >5% errors
}

func (e *ExternalFailureDetector) detectRetryStorm(ctx context.Context, serviceName string) bool {
	reqMetrics, err := e.db.GetRecentMetrics(ctx, serviceName, "request_rate", 10*time.Minute)
	if err != nil || len(reqMetrics) < 5 {
		return false
	}

	// Check for sudden request rate spike (3x normal)
	if len(reqMetrics) < 6 {
		return false
	}

	mid := len(reqMetrics) / 2
	first := CalculateAverage(reqMetrics[:mid])
	second := CalculateAverage(reqMetrics[mid:])

	return second > first*3.0 && first > 0
}

func (e *ExternalFailureDetector) analyzeExternalCorrelation(ctx context.Context, serviceName string) float64 {
	errorMetrics, err := e.db.GetRecentMetrics(ctx, serviceName, "error_rate", 15*time.Minute)
	if err != nil || len(errorMetrics) < 5 {
		return 0
	}

	respMetrics, err := e.db.GetRecentMetrics(ctx, serviceName, "response_time", 15*time.Minute)
	if err != nil || len(respMetrics) < 5 {
		return 0
	}

	// Calculate Pearson correlation
	return CalculatePearsonCorrelation(errorMetrics, respMetrics)
}

func (e *ExternalFailureDetector) detectResourceErrorMismatch(ctx context.Context, serviceName string) bool {
	errorMetrics, err := e.db.GetRecentMetrics(ctx, serviceName, "error_rate", 10*time.Minute)
	if err != nil || len(errorMetrics) < 3 {
		return false
	}

	cpuMetrics, err := e.db.GetRecentMetrics(ctx, serviceName, "cpu_usage", 10*time.Minute)
	if err != nil || len(cpuMetrics) < 3 {
		return false
	}

	avgErr := CalculateAverage(errorMetrics)
	avgCPU := CalculateAverage(cpuMetrics)

	// Mismatch: high errors but low CPU usage
	return avgErr > 5.0 && avgCPU < 50.0
}

func (e *ExternalFailureDetector) buildRecommendation(detected bool, severity string, timeout, retryStorm bool) string {
	if !detected {
		return "No external dependency failures detected."
	}

	rec := ""
	if severity == "CRITICAL" {
		rec = "CRITICAL EXTERNAL FAILURE: Immediate investigation required. "
	} else {
		rec = "EXTERNAL FAILURE WARNING: "
	}

	if timeout {
		rec += "Timeout patterns detected - check external service health. "
	}
	if retryStorm {
		rec += "Retry storm detected - implement circuit breaker. "
	}

	rec += "Actions: 1) Check external service status. 2) Review timeout configurations. 3) Implement circuit breakers. 4) Add fallback mechanisms."

	return rec
}

func (e *ExternalFailureDetector) calculateSeverity(confidence float64, timeout, retryStorm bool) string {
	if confidence < 70 {
		return "LOW"
	}
	if timeout && retryStorm {
		return "CRITICAL"
	}
	if timeout || retryStorm {
		return "HIGH"
	}
	return "MEDIUM"
}
