package analyzer

import (
	"context"
	"sort"
	"time"

	"github.com/namansh70747/AURA-Autonomous-Unified-Reliability-Automation-Platform/internal/storage"
	"github.com/namansh70747/AURA-Autonomous-Unified-Reliability-Automation-Platform/pkg/logger"
	"go.uber.org/zap"
)

type Analyzer struct {
	db                         *storage.PostgresClient
	memoryLeakDetector         *MemoryLeakDetector
	deploymentBugDetector      *DeploymentBugDetector
	cascadeDetector            *CascadeDetector
	externalFailureDetector    *ExternalFailureDetector
	resourceExhaustionDetector *ResourceExhaustionDetector
}

func NewAnalyzer(db *storage.PostgresClient) *Analyzer {
	logger.Info("Initializing pattern analyzer")

	return &Analyzer{
		db:                         db,
		memoryLeakDetector:         NewMemoryLeakDetector(db),
		deploymentBugDetector:      NewDeploymentBugDetector(db),
		cascadeDetector:            NewCascadeDetector(db),
		externalFailureDetector:    NewExternalFailureDetector(db),
		resourceExhaustionDetector: NewResourceExhaustionDetector(db),
	}
}

func (a *Analyzer) AnalyzeService(ctx context.Context, serviceName string) (*Diagnosis, error) {
	logger.Info("Starting pattern analysis",
		zap.String("service", serviceName),
	)

	results := make(chan *Detection, 5)
	errors := make(chan error, 5)

	go func() {
		detection, err := a.memoryLeakDetector.Analyze(ctx, serviceName)
		if err != nil {
			errors <- err
			return
		}
		results <- detection
	}()

	go func() {
		detection, err := a.deploymentBugDetector.Analyze(ctx, serviceName)
		if err != nil {
			errors <- err
			return
		}
		results <- detection
	}()

	go func() {
		detection, err := a.cascadeDetector.Analyze(ctx, serviceName)
		if err != nil {
			errors <- err
			return
		}
		results <- detection
	}()

	go func() {
		detection, err := a.externalFailureDetector.Analyze(ctx, serviceName)
		if err != nil {
			errors <- err
			return
		}
		results <- detection
	}()

	go func() {
		detection, err := a.resourceExhaustionDetector.Analyze(ctx, serviceName)
		if err != nil {
			errors <- err
			return
		}
		results <- detection
	}()

	detections := []*Detection{}
	for i := 0; i < 5; i++ {
		select {
		case detection := <-results:
			detections = append(detections, detection)
			logger.Debug("Detection completed",
				zap.String("service", serviceName),
				zap.String("type", string(detection.Type)),
				zap.Bool("detected", detection.Detected),
				zap.Float64("confidence", detection.Confidence),
			)
		case err := <-errors:
			logger.Error("Detection failed",
				zap.String("service", serviceName),
				zap.Error(err),
			)
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	sort.Slice(detections, func(i, j int) bool {
		return detections[i].Confidence > detections[j].Confidence
	})

	bestMatch := detections[0]

	diagnosis := &Diagnosis{
		ServiceName:         serviceName,
		Problem:             DetectionHealthy,
		Confidence:          0,
		Timestamp:           time.Now(),
		Evidence:            map[string]interface{}{},
		Recommendation:      "Service is healthy. No issues detected.",
		Severity:            "LOW",
		AllDetections:       make([]Detection, 0),
		MultipleProblems:    false,
		HighConfidenceCount: 0,
	}

	for _, d := range detections {
		diagnosis.AllDetections = append(diagnosis.AllDetections, *d)
	}

	// Find ALL problems with confidence > 80% (not just the best match)
	highConfidenceDetections := []*Detection{}
	for _, d := range detections {
		if d.Detected && d.Confidence > 80.0 {
			highConfidenceDetections = append(highConfidenceDetections, d)
		}
	}

	if len(highConfidenceDetections) > 0 {
		// Use best match for main diagnosis
		diagnosis.Problem = bestMatch.Type
		diagnosis.Confidence = bestMatch.Confidence
		diagnosis.Evidence = bestMatch.Evidence
		diagnosis.Recommendation = bestMatch.Recommendation
		diagnosis.Severity = bestMatch.Severity
		diagnosis.HighConfidenceCount = len(highConfidenceDetections)
		diagnosis.MultipleProblems = len(highConfidenceDetections) > 1

		// If multiple problems, add note to recommendation
		if diagnosis.MultipleProblems {
			problemList := ""
			for i, d := range highConfidenceDetections {
				if i > 0 {
					problemList += ", "
				}
				problemList += string(d.Type)
			}
			diagnosis.Recommendation = diagnosis.Recommendation +
				" | ALERT: Multiple issues detected (" + problemList + "). Check all_detections for details."
		}

		// Log all high-confidence problems
		if len(highConfidenceDetections) > 1 {
			logger.Warn("Multiple problems detected",
				zap.String("service", serviceName),
				zap.Int("problem_count", len(highConfidenceDetections)),
			)
		}

		// Save ALL high-confidence detections to database
		if a.db != nil {
			savedCount := 0
			for _, detection := range highConfidenceDetections {
				diagnosisRecord := &storage.DiagnosisRecord{
					ServiceName:    serviceName,
					ProblemType:    string(detection.Type),
					Confidence:     detection.Confidence,
					Severity:       detection.Severity,
					Evidence:       detection.Evidence,
					Recommendation: detection.Recommendation,
					Timestamp:      time.Now(),
				}

				if err := a.db.SaveDiagnosis(ctx, diagnosisRecord); err != nil {
					logger.Error("Failed to save diagnosis",
						zap.String("problem", string(detection.Type)),
						zap.Error(err),
					)
				} else {
					savedCount++
				}
			}

			logger.Info("Diagnoses saved",
				zap.String("service", serviceName),
				zap.Int("saved_count", savedCount),
				zap.Int("total_detected", len(highConfidenceDetections)),
			)
		}

		// Log primary problem
		logger.Info("Problem detected",
			zap.String("service", serviceName),
			zap.String("primary_problem", string(bestMatch.Type)),
			zap.Float64("confidence", bestMatch.Confidence),
			zap.String("severity", bestMatch.Severity),
			zap.Int("total_problems", len(highConfidenceDetections)),
		)
	} else {
		logger.Info("Service healthy",
			zap.String("service", serviceName),
			zap.Float64("highest_confidence", bestMatch.Confidence),
		)
	}

	return diagnosis, nil
}

func (a *Analyzer) AnalyzeAllServices(ctx context.Context, services []string) (map[string]*Diagnosis, error) {
	logger.Info("Analyzing all services",
		zap.Int("count", len(services)),
	)

	results := make(map[string]*Diagnosis)

	for _, service := range services {
		diagnosis, err := a.AnalyzeService(ctx, service)
		if err != nil {
			logger.Error("Failed to analyze service",
				zap.String("service", service),
				zap.Error(err),
			)
			continue
		}
		results[service] = diagnosis
	}

	return results, nil
}
