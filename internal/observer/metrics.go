package observer

import (
	"context"
	"fmt"
	"time"

	"github.com/namansh70747/AURA-Autonomous-Unified-Reliability-Automation-Platform/internal/storage"
	"go.uber.org/zap"
)

type MetricsObserver struct {
	prometheus *PrometheusClient
	kubernetes *KubernetesWatcher
	db         *storage.PostgresClient
	logger     *zap.Logger
}

func NewMetricsObserver(
	prometheusURL string,
	scrapeInterval time.Duration,
	k8sNamespace string,
	db *storage.PostgresClient,
	logger *zap.Logger,
) (*MetricsObserver, error) {
	promClient, err := NewPrometheusClient(prometheusURL, scrapeInterval, db, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create prometheus client: %w", err)
	}

	k8sWatcher, err := NewKubernetesWatcher(k8sNamespace, db, logger)
	if err != nil {
		logger.Warn("Kubernetes watcher not available", zap.Error(err))
		k8sWatcher = nil
	}

	return &MetricsObserver{
		prometheus: promClient,
		kubernetes: k8sWatcher,
		db:         db,
		logger:     logger,
	}, nil
}

func (m *MetricsObserver) Start(ctx context.Context) error {
	go func() {
		if err := m.prometheus.Start(ctx); err != nil && err != context.Canceled {
			m.logger.Error("Prometheus error", zap.Error(err))
		}
	}()

	if m.kubernetes != nil {
		go func() {
			if err := m.kubernetes.Start(ctx); err != nil && err != context.Canceled {
				m.logger.Error("Kubernetes error", zap.Error(err))
			}
		}()
	}

	<-ctx.Done()
	return nil
}

func (m *MetricsObserver) GetCurrentMetrics(ctx context.Context, serviceName string) (*ServiceMetrics, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	metrics := &ServiceMetrics{
		ServiceName: serviceName,
		Timestamp:   time.Now(),
	}

	cpuMetrics, err := m.db.GetRecentMetrics(ctx, serviceName, "cpu_usage", 1*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("failed to get cpu metrics: %w", err)
	}
	if len(cpuMetrics) > 0 {
		metrics.CPUUsage = cpuMetrics[0].MetricValue
	}

	memMetrics, err := m.db.GetRecentMetrics(ctx, serviceName, "memory_usage", 1*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("failed to get memory metrics: %w", err)
	}
	if len(memMetrics) > 0 {
		metrics.MemoryUsage = memMetrics[0].MetricValue
	}

	requestMetrics, err := m.db.GetRecentMetrics(ctx, serviceName, "http_requests", 5*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("failed to get request metrics: %w", err)
	}
	metrics.RequestCount = int64(len(requestMetrics))

	errorMetrics, err := m.db.GetRecentMetrics(ctx, serviceName, "error_count", 5*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("failed to get error metrics: %w", err)
	}
	metrics.ErrorCount = int64(len(errorMetrics))

	if metrics.RequestCount > 0 {
		metrics.ErrorRate = (float64(metrics.ErrorCount) / float64(metrics.RequestCount)) * 100
	}

	return metrics, nil
}

func (m *MetricsObserver) Health(ctx context.Context) error {
	if err := m.prometheus.Health(ctx); err != nil {
		return fmt.Errorf("prometheus health check failed: %w", err)
	}

	if m.kubernetes != nil {
		if err := m.kubernetes.Health(ctx); err != nil {
			m.logger.Warn("Kubernetes health check failed", zap.Error(err))
		}
	}

	if err := m.db.Health(ctx); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	return nil
}

type ServiceMetrics struct {
	ServiceName  string    `json:"service_name"`
	Timestamp    time.Time `json:"timestamp"`
	CPUUsage     float64   `json:"cpu_usage"`
	MemoryUsage  float64   `json:"memory_usage"`
	RequestCount int64     `json:"request_count"`
	ErrorCount   int64     `json:"error_count"`
	ErrorRate    float64   `json:"error_rate"`
	Latency      float64   `json:"latency_ms"`
}

func (s *ServiceMetrics) IsHealthy(cpuThreshold, memThreshold, errorRateThreshold float64) bool {
	if s.CPUUsage > cpuThreshold {
		return false
	}
	if s.MemoryUsage > memThreshold {
		return false
	}
	if s.ErrorRate > errorRateThreshold {
		return false
	}
	return true
}

func (m *MetricsObserver) GetKubernetesPods(ctx context.Context) ([]PodMetric, error) {
	if m.kubernetes == nil {
		return nil, fmt.Errorf("kubernetes watcher not initialized")
	}
	return m.kubernetes.GetPodMetrics(ctx)
}
