package observer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/namansh70747/AURA-Autonomous-Unified-Reliability-Automation-Platform/internal/storage"
	promapi "github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"go.uber.org/zap"
)

type PrometheusClient struct {
	client   promapi.Client
	api      promv1.API
	url      string
	interval time.Duration
	db       *storage.PostgresClient
	logger   *zap.Logger
}

func NewPrometheusClient(prometheusURL string, scrapeInterval time.Duration, db *storage.PostgresClient, logger *zap.Logger) (*PrometheusClient, error) {
	client, err := promapi.NewClient(promapi.Config{
		Address: prometheusURL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create prometheus client: %w", err)
	}

	return &PrometheusClient{
		client:   client,
		api:      promv1.NewAPI(client),
		url:      prometheusURL,
		interval: scrapeInterval,
		db:       db,
		logger:   logger,
	}, nil
}

func (p *PrometheusClient) Start(ctx context.Context) error {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	if err := p.scrapeAllMetrics(ctx); err != nil {
		p.logger.Error("Initial metric scrape failed", zap.Error(err))
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := p.scrapeAllMetrics(ctx); err != nil {
				p.logger.Error("Metric scrape failed", zap.Error(err))
			}
		}
	}
}

func (p *PrometheusClient) scrapeAllMetrics(ctx context.Context) error {
	metrics := []struct {
		query      string
		metricName string
	}{
		{"cpu_usage_percent", "cpu_usage"},
		{"memory_usage_percent", "memory_usage"},
		{"http_requests_total", "http_requests"},
		{"http_request_duration_seconds", "http_latency"},
		{"app_errors_total", "error_count"},
	}

	var collectedMetrics []*storage.Metric
	timestamp := time.Now()

	for _, m := range metrics {
		result, err := p.queryMetric(ctx, m.query)
		if err != nil {
			p.logger.Warn("Failed to query metric",
				zap.String("metric", m.metricName),
				zap.Error(err),
			)
			continue
		}

		for _, sample := range result {
			metric := &storage.Metric{
				Timestamp:   timestamp,
				ServiceName: string(sample.Metric["service"]),
				MetricName:  m.metricName,
				MetricValue: float64(sample.Value),
				Labels:      marshalPromLabels(sample.Metric),
			}

			if metric.ServiceName == "" {
				metric.ServiceName = "sample-app"
			}

			collectedMetrics = append(collectedMetrics, metric)
		}
	}

	if len(collectedMetrics) > 0 {
		if err := p.db.BatchSaveMetrics(ctx, collectedMetrics); err != nil {
			return fmt.Errorf("failed to save metrics batch: %w", err)
		}
	}

	return nil
}

func (p *PrometheusClient) queryMetric(ctx context.Context, query string) (model.Vector, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	result, warnings, err := p.api.Query(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("prometheus query failed: %w", err)
	}
	if len(warnings) > 0 {
		p.logger.Warn("Prometheus query warnings",
			zap.Strings("warnings", warnings),
		)
	}

	vector, ok := result.(model.Vector)
	if !ok {
		return nil, fmt.Errorf("unexpected result type: %T", result)
	}

	return vector, nil
}

func (p *PrometheusClient) Health(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, _, err := p.api.Query(ctx, "up", time.Now())
	if err != nil {
		return fmt.Errorf("prometheus health check failed: %w", err)
	}

	return nil
}

func marshalPromLabels(metric model.Metric) []byte {
	labels := make(map[string]string)
	for k, v := range metric {
		labels[string(k)] = string(v)
	}
	data, err := json.Marshal(labels)
	if err != nil {
		return []byte("{}")
	}
	return data
}
