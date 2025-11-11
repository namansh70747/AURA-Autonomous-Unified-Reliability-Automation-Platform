package analyzer

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/namansh70747/AURA-Autonomous-Unified-Reliability-Automation-Platform/pkg/logger"
	"go.uber.org/zap"
)

// EnhancedDetector uses feature-based multi-signal detection
type EnhancedDetector struct {
	featureExtractor *FeatureExtractor
}

func NewEnhancedDetector(fe *FeatureExtractor) *EnhancedDetector {
	return &EnhancedDetector{
		featureExtractor: fe,
	}
}

// DetectMemoryLeakEnhanced uses improved 6-signal approach with quality gating
func (ed *EnhancedDetector) DetectMemoryLeakEnhanced(ctx context.Context, serviceName string) (*Detection, error) {
	features, err := ed.featureExtractor.ExtractFeatures(ctx, serviceName, 30*time.Minute)
	if err != nil {
		return nil, err
	}

	signals := make(map[string]float64)
	signalQuality := 0 // Count of high-quality signals

	// Signal 1: Positive memory trend with minimum threshold (35% weight)
	// IMPROVED: Only trigger if trend is sustained (> 0.15% per minute)
	if features.MemoryTrend > 0.15 {
		trendScore := math.Min(100, features.MemoryTrend*15) * 0.35
		signals["trend"] = trendScore
		if trendScore > 25 { // High quality signal
			signalQuality++
		}
		logger.Debug("Memory leak signal: trend",
			zap.Float64("trend", features.MemoryTrend),
			zap.Float64("score", trendScore))
	}

	// Signal 2: Low volatility + positive trend = sustained growth (25% weight)
	// IMPROVED: Require both low volatility AND positive trend
	if features.MemoryVolatility < 0.15 && features.MemoryTrend > 0.1 {
		volatilityScore := (1 - features.MemoryVolatility) * 100 * 0.25
		signals["low_volatility"] = volatilityScore
		if volatilityScore > 20 {
			signalQuality++
		}
		logger.Debug("Memory leak signal: low volatility",
			zap.Float64("volatility", features.MemoryVolatility),
			zap.Float64("score", volatilityScore))
	}

	// Signal 3: High memory level (20% weight)
	// IMPROVED: Only count if memory is dangerously high (> 75%)
	if features.MemoryMean > 75 {
		levelScore := ((features.MemoryMean - 75) / 25) * 100 * 0.20
		signals["level"] = levelScore
		if features.MemoryMean > 85 {
			signalQuality++
		}
	}

	// Signal 4: Growing range indicates increasing baseline (10% weight)
	// IMPROVED: Require minimum range of 15%
	if features.MemoryRange > 15 {
		rangeScore := math.Min((features.MemoryRange/50)*100, 100) * 0.10
		signals["range"] = rangeScore
	}

	// Signal 5: High autocorrelation = persistent pattern (10% weight)
	// IMPROVED: Require very high autocorrelation (> 0.8)
	if features.MemoryAutocorrelation > 0.8 {
		autocorrScore := features.MemoryAutocorrelation * 100 * 0.10
		signals["autocorr"] = autocorrScore
		signalQuality++
	}

	// NEW Signal 6: Cross-validation - NO correlation with CPU (bonus)
	// Real memory leaks don't correlate with CPU usage
	if math.Abs(features.CPUMemoryCorr) < 0.3 && features.MemoryTrend > 0.1 {
		signals["independent_growth"] = 15.0 // Bonus signal
		signalQuality++
		logger.Debug("Memory leak signal: independent growth detected",
			zap.Float64("cpu_memory_corr", features.CPUMemoryCorr))
	}

	// Aggregate confidence with quality gating
	totalConfidence := 0.0
	for _, conf := range signals {
		totalConfidence += conf
	}

	// IMPROVED: Require at least 2 high-quality signals AND minimum confidence
	detected := totalConfidence > 65 && signalQuality >= 2

	// Dampen confidence if we have weak signals
	if signalQuality < 2 {
		totalConfidence *= 0.7 // Reduce confidence by 30%
	}

	severity := SeverityNone
	if detected {
		if totalConfidence > 90 && signalQuality >= 3 {
			severity = SeverityCritical
		} else if totalConfidence > 80 {
			severity = SeverityHigh
		} else if totalConfidence > 70 {
			severity = SeverityMedium
		} else {
			severity = SeverityLow
		}
	}

	evidence := map[string]interface{}{
		"memory_trend":      fmt.Sprintf("%.4f%%/min", features.MemoryTrend),
		"current_memory":    fmt.Sprintf("%.2f%%", features.MemoryMean),
		"memory_range":      fmt.Sprintf("%.2f%%", features.MemoryRange),
		"autocorrelation":   fmt.Sprintf("%.3f", features.MemoryAutocorrelation),
		"volatility":        fmt.Sprintf("%.3f", features.MemoryVolatility),
		"cpu_memory_corr":   fmt.Sprintf("%.3f", features.CPUMemoryCorr),
		"signals":           signals,
		"signal_quality":    signalQuality,
		"total_signals":     len(signals),
		"quality_gate_pass": signalQuality >= 2,
	}

	if detected && features.MemoryTrend > 0 {
		remainingCapacity := 100 - features.MemoryMean
		minutesToFull := remainingCapacity / features.MemoryTrend
		evidence["time_to_exhaustion_min"] = fmt.Sprintf("%.1f", minutesToFull)
		evidence["estimated_oom"] = time.Now().Add(time.Duration(minutesToFull) * time.Minute).Format(time.RFC3339)
	}

	recommendation := "No action required"
	if detected {
		switch severity {
		case SeverityCritical:
			recommendation = "🚨 IMMEDIATE: Memory leak confirmed. Restart pod or increase memory limit NOW."
		case SeverityHigh:
			recommendation = "⚠️  URGENT: Strong memory leak indicators. Monitor closely and prepare restart."
		default:
			recommendation = "📊 Possible memory leak pattern. Continue monitoring for 15 more minutes."
		}
	}

	logger.Info("Memory leak detection complete",
		zap.String("service", serviceName),
		zap.Bool("detected", detected),
		zap.Float64("confidence", totalConfidence),
		zap.Int("signal_quality", signalQuality))

	return &Detection{
		Type:           DetectionMemoryLeak,
		ServiceName:    serviceName,
		Detected:       detected,
		Confidence:     totalConfidence,
		Severity:       severity,
		Evidence:       evidence,
		Recommendation: recommendation,
		Timestamp:      time.Now(),
	}, nil
}

// DetectResourceExhaustionEnhanced with improved thresholds
func (ed *EnhancedDetector) DetectResourceExhaustionEnhanced(ctx context.Context, serviceName string) (*Detection, error) {
	features, err := ed.featureExtractor.ExtractFeatures(ctx, serviceName, 15*time.Minute)
	if err != nil {
		return nil, err
	}

	signals := make(map[string]float64)
	signalQuality := 0

	// Signal 1: High CPU (30% weight)
	// IMPROVED: Require sustained high CPU (> 80%)
	if features.CPUMean > 80 {
		cpuScore := ((features.CPUMean - 80) / 20) * 100 * 0.30
		signals["cpu_high"] = cpuScore
		if features.CPUMean > 90 {
			signalQuality++
		}
	}

	// Signal 2: High Memory (30% weight)
	// IMPROVED: Require sustained high memory (> 85%)
	if features.MemoryMean > 85 {
		memScore := ((features.MemoryMean - 85) / 15) * 100 * 0.30
		signals["memory_high"] = memScore
		if features.MemoryMean > 92 {
			signalQuality++
		}
	}

	// Signal 3: Rising errors (25% weight)
	// IMPROVED: Require error rate AND error trend
	if features.ErrorRateMean > 8 || features.ErrorRateTrend > 2 {
		errorScore := math.Min((features.ErrorRateMean/15)*100, 100) * 0.25
		signals["errors"] = errorScore
		if features.ErrorRateMean > 15 {
			signalQuality++
		}
	}

	// Signal 4: System stress (15% weight)
	if features.SystemStress > 75 {
		stressScore := ((features.SystemStress - 75) / 25) * 100 * 0.15
		signals["stress"] = stressScore
		if features.SystemStress > 85 {
			signalQuality++
		}
	}

	// IMPROVED: Require BOTH high CPU AND high memory for exhaustion
	// This prevents false positives from single resource spikes
	bothHigh := features.CPUMean > 80 && features.MemoryMean > 80
	if bothHigh {
		signals["both_resources_high"] = 20.0 // Bonus
		signalQuality++
	}

	totalConfidence := 0.0
	for _, conf := range signals {
		totalConfidence += conf
	}

	// IMPROVED: Higher threshold and require quality signals
	detected := totalConfidence > 60 && (signalQuality >= 2 || bothHigh)

	if signalQuality < 2 && !bothHigh {
		totalConfidence *= 0.75
	}

	severity := SeverityNone
	if detected {
		if totalConfidence > 85 && bothHigh {
			severity = SeverityCritical
		} else if totalConfidence > 75 {
			severity = SeverityHigh
		} else if totalConfidence > 65 {
			severity = SeverityMedium
		} else {
			severity = SeverityLow
		}
	}

	evidence := map[string]interface{}{
		"cpu_usage":      fmt.Sprintf("%.2f%%", features.CPUMean),
		"memory_usage":   fmt.Sprintf("%.2f%%", features.MemoryMean),
		"error_rate":     fmt.Sprintf("%.2f/min", features.ErrorRateMean),
		"system_stress":  fmt.Sprintf("%.2f/100", features.SystemStress),
		"health_score":   fmt.Sprintf("%.2f/100", features.HealthScore),
		"both_high":      bothHigh,
		"signals":        signals,
		"signal_quality": signalQuality,
	}

	recommendation := "No action required"
	if detected {
		switch severity {
		case SeverityCritical:
			recommendation = "🚨 CRITICAL: Both CPU and Memory exhausted. Scale up immediately or increase limits."
		case SeverityHigh:
			recommendation = "⚠️  Scale horizontally (add replicas) or vertically (increase limits) soon."
		default:
			recommendation = "📊 Resource usage elevated. Monitor and prepare scaling plan."
		}
	}

	logger.Info("Resource exhaustion detection complete",
		zap.String("service", serviceName),
		zap.Bool("detected", detected),
		zap.Float64("confidence", totalConfidence),
		zap.Bool("both_resources_high", bothHigh))

	return &Detection{
		Type:           DetectionResourceExhaustion,
		ServiceName:    serviceName,
		Detected:       detected,
		Confidence:     totalConfidence,
		Severity:       severity,
		Evidence:       evidence,
		Recommendation: recommendation,
		Timestamp:      time.Now(),
	}, nil
}

// DetectDeploymentBugEnhanced with better correlation analysis
func (ed *EnhancedDetector) DetectDeploymentBugEnhanced(ctx context.Context, serviceName string) (*Detection, error) {
	features, err := ed.featureExtractor.ExtractFeatures(ctx, serviceName, 20*time.Minute)
	if err != nil {
		return nil, err
	}

	signals := make(map[string]float64)
	signalQuality := 0

	// Signal 1: Sudden error spike (40% weight)
	// IMPROVED: Require BOTH high spikiness AND high error rate
	if features.ErrorRateSpikiness > 2.0 && features.ErrorRateMean > 5 {
		spikeScore := math.Min(features.ErrorRateSpikiness*20, 100) * 0.40
		signals["error_spike"] = spikeScore
		if features.ErrorRateSpikiness > 3.0 {
			signalQuality++
		}
	}

	// Signal 2: High error rate (25% weight)
	if features.ErrorRateMean > 15 {
		rateScore := math.Min((features.ErrorRateMean/40)*100, 100) * 0.25
		signals["error_rate"] = rateScore
		if features.ErrorRateMean > 25 {
			signalQuality++
		}
	}

	// Signal 3: Errors independent of load (20% weight)
	// IMPROVED: Stricter threshold for independence
	if math.Abs(features.CPUErrorCorr) < 0.25 && features.ErrorRateMean > 10 {
		indepScore := (1 - math.Abs(features.CPUErrorCorr)) * 100 * 0.20
		signals["independent_errors"] = indepScore
		signalQuality++
	}

	// Signal 4: Instability (15% weight)
	if features.StabilityIndex < 4 {
		instabilityScore := ((4 - features.StabilityIndex) / 4) * 100 * 0.15
		signals["instability"] = instabilityScore
	}

	// NEW: Cross-validate with resource usage
	// Deployment bugs typically DON'T cause high resource usage
	normalResources := features.CPUMean < 70 && features.MemoryMean < 70
	if normalResources && features.ErrorRateMean > 10 {
		signals["normal_resources_high_errors"] = 15.0 // Bonus
		signalQuality++
	}

	totalConfidence := 0.0
	for _, conf := range signals {
		totalConfidence += conf
	}

	// IMPROVED: Require minimum signal quality
	detected := totalConfidence > 55 && signalQuality >= 2

	if signalQuality < 2 {
		totalConfidence *= 0.70
	}

	severity := SeverityNone
	if detected {
		if totalConfidence > 80 && signalQuality >= 3 {
			severity = SeverityCritical
		} else if totalConfidence > 70 {
			severity = SeverityHigh
		} else {
			severity = SeverityMedium
		}
	}

	evidence := map[string]interface{}{
		"error_rate":       fmt.Sprintf("%.2f/min", features.ErrorRateMean),
		"error_spikiness":  fmt.Sprintf("%.2f", features.ErrorRateSpikiness),
		"cpu_error_corr":   fmt.Sprintf("%.3f", features.CPUErrorCorr),
		"stability_index":  fmt.Sprintf("%.2f/10", features.StabilityIndex),
		"cpu_mean":         fmt.Sprintf("%.2f%%", features.CPUMean),
		"memory_mean":      fmt.Sprintf("%.2f%%", features.MemoryMean),
		"normal_resources": normalResources,
		"signals":          signals,
		"signal_quality":   signalQuality,
	}

	recommendation := "No action required"
	if detected {
		switch severity {
		case SeverityCritical:
			recommendation = "🚨 ROLLBACK: Deployment bug detected with high confidence. Rollback immediately."
		case SeverityHigh:
			recommendation = "⚠️  Likely deployment bug. Investigate recent changes and prepare rollback."
		default:
			recommendation = "📊 Possible deployment issue. Review error logs and recent deployments."
		}
	}

	logger.Info("Deployment bug detection complete",
		zap.String("service", serviceName),
		zap.Bool("detected", detected),
		zap.Float64("confidence", totalConfidence),
		zap.Bool("normal_resources", normalResources))

	return &Detection{
		Type:           DetectionDeploymentBug,
		ServiceName:    serviceName,
		Detected:       detected,
		Confidence:     totalConfidence,
		Severity:       severity,
		Evidence:       evidence,
		Recommendation: recommendation,
		Timestamp:      time.Now(),
	}, nil
}

// DetectExternalFailureEnhanced with better pattern matching
func (ed *EnhancedDetector) DetectExternalFailureEnhanced(ctx context.Context, serviceName string) (*Detection, error) {
	features, err := ed.featureExtractor.ExtractFeatures(ctx, serviceName, 15*time.Minute)
	if err != nil {
		return nil, err
	}

	signals := make(map[string]float64)
	signalQuality := 0

	// Signal 1: High latency (35% weight)
	// IMPROVED: Use P99 instead of P95 for external failures
	if features.LatencyP99 > 3000 {
		latencyScore := math.Min((features.LatencyP99-3000)/10000*100, 100) * 0.35
		signals["latency"] = latencyScore
		if features.LatencyP99 > 5000 {
			signalQuality++
		}
	}

	// Signal 2: Strong Latency-Error correlation (30% weight)
	// IMPROVED: Require strong correlation (> 0.6)
	if math.Abs(features.LatencyErrorCorr) > 0.6 {
		corrScore := math.Abs(features.LatencyErrorCorr) * 100 * 0.30
		signals["latency_error_corr"] = corrScore
		signalQuality++
	}

	// Signal 3: High errors with LOW internal resource usage (20% weight)
	// IMPROVED: This is the KEY signal for external failures
	if features.ErrorRateMean > 10 && features.CPUMean < 65 && features.MemoryMean < 70 {
		externalScore := (features.ErrorRateMean / 30) * 100 * 0.20
		signals["external_pattern"] = externalScore
		signalQuality++
	}

	// Signal 4: Error spikiness (15% weight)
	if features.ErrorRateSpikiness > 2.5 {
		spikeScore := math.Min(features.ErrorRateSpikiness*20, 100) * 0.15
		signals["error_spikes"] = spikeScore
	}

	// NEW: Cross-validation - Memory-Error correlation should be LOW
	// External failures don't correlate with memory
	if math.Abs(features.MemoryErrorCorr) < 0.3 && features.ErrorRateMean > 8 {
		signals["no_memory_correlation"] = 10.0 // Bonus
		signalQuality++
	}

	totalConfidence := 0.0
	for _, conf := range signals {
		totalConfidence += conf
	}

	// IMPROVED: Require the "external pattern" signal for detection
	hasExternalPattern := features.ErrorRateMean > 10 && features.CPUMean < 65
	detected := totalConfidence > 55 && (hasExternalPattern || signalQuality >= 3)

	if !hasExternalPattern && signalQuality < 3 {
		totalConfidence *= 0.65
	}

	severity := SeverityNone
	if detected {
		if totalConfidence > 85 && hasExternalPattern {
			severity = SeverityCritical
		} else if totalConfidence > 75 {
			severity = SeverityHigh
		} else {
			severity = SeverityMedium
		}
	}

	evidence := map[string]interface{}{
		"latency_p99":        fmt.Sprintf("%.2fms", features.LatencyP99),
		"latency_p95":        fmt.Sprintf("%.2fms", features.LatencyP95),
		"error_rate":         fmt.Sprintf("%.2f/min", features.ErrorRateMean),
		"latency_error_corr": fmt.Sprintf("%.3f", features.LatencyErrorCorr),
		"cpu_usage":          fmt.Sprintf("%.2f%%", features.CPUMean),
		"memory_usage":       fmt.Sprintf("%.2f%%", features.MemoryMean),
		"external_pattern":   hasExternalPattern,
		"signals":            signals,
		"signal_quality":     signalQuality,
	}

	recommendation := "No action required"
	if detected {
		switch severity {
		case SeverityCritical:
			recommendation = "🚨 External dependency failure detected. Check databases, APIs, and network. Enable fallbacks."
		case SeverityHigh:
			recommendation = "⚠️  Likely external issue. Monitor downstream services and implement retry/timeout policies."
		default:
			recommendation = "📊 Investigate external dependencies. Check third-party service status."
		}
	}

	logger.Info("External failure detection complete",
		zap.String("service", serviceName),
		zap.Bool("detected", detected),
		zap.Float64("confidence", totalConfidence),
		zap.Bool("external_pattern", hasExternalPattern))

	return &Detection{
		Type:           DetectionExternalFailure,
		ServiceName:    serviceName,
		Detected:       detected,
		Confidence:     totalConfidence,
		Severity:       severity,
		Evidence:       evidence,
		Recommendation: recommendation,
		Timestamp:      time.Now(),
	}, nil
}

// DetectCascadeFailureEnhanced with system-wide analysis
func (ed *EnhancedDetector) DetectCascadeFailureEnhanced(ctx context.Context, serviceName string) (*Detection, error) {
	features, err := ed.featureExtractor.ExtractFeatures(ctx, serviceName, 20*time.Minute)
	if err != nil {
		return nil, err
	}

	signals := make(map[string]float64)
	signalQuality := 0

	// Signal 1: Multiple resource degradation (35% weight)
	// IMPROVED: Count degraded resources more carefully
	degradedCount := 0
	degradationSeverity := 0.0

	if features.CPUMean > 85 {
		degradedCount++
		degradationSeverity += (features.CPUMean - 85) / 15
	}
	if features.MemoryMean > 88 {
		degradedCount++
		degradationSeverity += (features.MemoryMean - 88) / 12
	}
	if features.ErrorRateMean > 15 {
		degradedCount++
		degradationSeverity += features.ErrorRateMean / 50
	}
	if features.LatencyP95 > 2000 {
		degradedCount++
		degradationSeverity += (features.LatencyP95 - 2000) / 8000
	}

	if degradedCount >= 3 {
		multiScore := math.Min(degradationSeverity*100, 100) * 0.35
		signals["multi_degradation"] = multiScore
		signalQuality++
	}

	// Signal 2: Critical system stress (30% weight)
	if features.SystemStress > 80 {
		stressScore := ((features.SystemStress - 80) / 20) * 100 * 0.30
		signals["system_stress"] = stressScore
		if features.SystemStress > 90 {
			signalQuality++
		}
	}

	// Signal 3: Very low health score (20% weight)
	if features.HealthScore < 35 {
		healthScore := ((35 - features.HealthScore) / 35) * 100 * 0.20
		signals["health"] = healthScore
		signalQuality++
	}

	// Signal 4: Multiple increasing trends (15% weight)
	trendCount := 0
	if features.CPUTrend > 0.5 {
		trendCount++
	}
	if features.MemoryTrend > 0.5 {
		trendCount++
	}
	if features.ErrorRateTrend > 2 {
		trendCount++
	}

	if trendCount >= 2 {
		trendScore := (float64(trendCount) / 3) * 100 * 0.15
		signals["trends"] = trendScore
		signalQuality++
	}

	// NEW: Cascade indicator - rapidly degrading stability
	if features.StabilityIndex < 2.5 {
		instabilityScore := ((2.5 - features.StabilityIndex) / 2.5) * 100 * 0.10
		signals["instability"] = instabilityScore
		if features.StabilityIndex < 1.5 {
			signalQuality++
		}
	}

	totalConfidence := 0.0
	for _, conf := range signals {
		totalConfidence += conf
	}

	// IMPROVED: Require multiple degraded resources AND quality signals
	detected := totalConfidence > 60 && degradedCount >= 3 && signalQuality >= 2

	if degradedCount < 3 || signalQuality < 2 {
		totalConfidence *= 0.75
	}

	severity := SeverityNone
	if detected {
		if totalConfidence > 85 && degradedCount >= 4 {
			severity = SeverityCritical
		} else if totalConfidence > 75 {
			severity = SeverityHigh
		} else {
			severity = SeverityMedium
		}
	}

	evidence := map[string]interface{}{
		"degraded_metrics": degradedCount,
		"system_stress":    fmt.Sprintf("%.2f/100", features.SystemStress),
		"health_score":     fmt.Sprintf("%.2f/100", features.HealthScore),
		"stability_index":  fmt.Sprintf("%.2f/10", features.StabilityIndex),
		"cpu_trend":        fmt.Sprintf("%.4f%%/min", features.CPUTrend),
		"memory_trend":     fmt.Sprintf("%.4f%%/min", features.MemoryTrend),
		"error_trend":      fmt.Sprintf("%.4f/min", features.ErrorRateTrend),
		"trending_metrics": trendCount,
		"signals":          signals,
		"signal_quality":   signalQuality,
	}

	recommendation := "No action required"
	if detected {
		switch severity {
		case SeverityCritical:
			recommendation = "🚨 CASCADE FAILURE: System-wide degradation. Scale entire cluster and isolate failing components immediately."
		case SeverityHigh:
			recommendation = "⚠️  Multiple systems degrading. Implement circuit breakers and scale proactively."
		default:
			recommendation = "📊 System stress increasing across multiple metrics. Monitor closely and prepare incident response."
		}
	}

	logger.Info("Cascade failure detection complete",
		zap.String("service", serviceName),
		zap.Bool("detected", detected),
		zap.Float64("confidence", totalConfidence),
		zap.Int("degraded_count", degradedCount),
		zap.Int("signal_quality", signalQuality))

	return &Detection{
		Type:           DetectionCascadingFailure,
		ServiceName:    serviceName,
		Detected:       detected,
		Confidence:     totalConfidence,
		Severity:       severity,
		Evidence:       evidence,
		Recommendation: recommendation,
		Timestamp:      time.Now(),
	}, nil
}
