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
	client   promapi.Client //Prometheus Client 
	api      promv1.API // Api of Prometheus 
	url      string // url we have of Prometheus 
	interval time.Duration // Type Time Interval 
	db       *storage.PostgresClient// db Postgres Client 
	logger   *zap.Logger// Logger 
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
}// new client with the given configuratiuon has started and then returned 

func (p *PrometheusClient) Start(ctx context.Context) error {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	if err := p.scrapeAllMetrics(ctx); err != nil {
		p.logger.Error("Initial metric scrape failed", zap.Error(err))
	}

	for { //infinite loop
		select { //select statement for context and ticker means it will wait for either the context to be done or the ticker to tick
		case <-ctx.Done(): //context is done
			return ctx.Err() //return the error because context is done and then error 
		case <-ticker.C: //ticker channel in easy language this is used to trigger events at regular intervals
			if err := p.scrapeAllMetrics(ctx); err != nil { // scrape all metrics
				p.logger.Error("Metric scrape failed", zap.Error(err))// if error occurs
			}
		}
	} //p.interval se time for ticker set kar diya hai and then we are scrapping all metrics at that interval
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
	} //array of strcut i have made 

	var collectedMetrics []*storage.Metric
	timestamp := time.Now() //we need it because we are using it as a timestamp for all metrics

	for _, m := range metrics {
		result, err := p.queryMetric(ctx, m.query) //model.vector of that query and then we are storing result 
		if err != nil { 
			p.logger.Warn("Failed to query metric",
				zap.String("metric", m.metricName),
				zap.Error(err),
			)
			continue //bahar niklo
		}

		for _, sample := range result {
			metric := &storage.Metric{
				Timestamp:   timestamp,
				ServiceName: string(sample.Metric["service"]),
				MetricName:  m.metricName,
				MetricValue: float64(sample.Value),
				Labels:      marshalPromLabels(sample.Metric),
			} //storage metric se. metric  name ki vastu bnai hai 

			if metric.ServiceName == "" {
				metric.ServiceName = "sample-app" //kuch nhi toh sample app hi sahi
			}

			collectedMetrics = append(collectedMetrics, metric)//append kardiya 
		}
	}// collected metrics ka array i have made and also 

	if len(collectedMetrics) > 0 {
		if err := p.db.BatchSaveMetrics(ctx, collectedMetrics); err != nil {
			return fmt.Errorf("failed to save metrics batch: %w", err)
		}
	} //Save kardiya Batch metrics ko 

	return nil
}

func (p *PrometheusClient) queryMetric(ctx context.Context, query string) (model.Vector, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	result, warnings, err := p.api.Query(ctx, query, time.Now()) // this is prometheus api call and query prom.api
	if err != nil {
		return nil, fmt.Errorf("prometheus query failed: %w", err)
	}
	if len(warnings) > 0 {
		p.logger.Warn("Prometheus query warnings",
			zap.Strings("warnings", warnings),
		)
	} //len of warning is greater than 0 than this will show the error 

	vector, ok := result.(model.Vector) // type assertion in easy language type assertion is used to convert interface type to specific type
	if !ok {
		return nil, fmt.Errorf("unexpected result type: %T", result) // maamla ok hai
	}

	return vector, nil //return the vector
} 

func (p *PrometheusClient) Health(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, _, err := p.api.Query(ctx, "up", time.Now())//up ki query hai, agar up hoga toh sab theek hai
	if err != nil {
		return fmt.Errorf("prometheus health check failed: %w", err)
	}

	return nil 
}//return kuch nhi hora bus health check ka endpoint bnaya hai promtheus ke liye 

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
