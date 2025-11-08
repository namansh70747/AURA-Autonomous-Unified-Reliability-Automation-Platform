package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type PostgresClient struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

func NewPostgresClient(connectionURL string, logger *zap.Logger) (*PostgresClient, error) {
	config, err := pgxpool.ParseConfig(connectionURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection URL: %w", err)
	}

	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute
	config.HealthCheckPeriod = 1 * time.Minute
	config.ConnConfig.ConnectTimeout = 10 * time.Second

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &PostgresClient{
		pool:   pool,
		logger: logger,
	}, nil
}

func (c *PostgresClient) Close() {
	c.pool.Close()
}

func (c *PostgresClient) Health(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return c.pool.Ping(ctx)
}

func (c *PostgresClient) SaveMetric(ctx context.Context, metric *Metric) error {
	query := `
		INSERT INTO metrics (timestamp, service_name, metric_name, metric_value, labels)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := c.pool.QueryRow(
		ctx,
		query,
		metric.Timestamp,
		metric.ServiceName,
		metric.MetricName,
		metric.MetricValue,
		metric.Labels,
	).Scan(&metric.ID)

	if err != nil {
		return fmt.Errorf("failed to save metric: %w", err)
	}

	return nil
}

func (c *PostgresClient) GetRecentMetrics(
	ctx context.Context,
	serviceName string,
	metricName string,
	duration time.Duration,
) ([]*Metric, error) {
	query := `
		SELECT id, timestamp, service_name, metric_name, metric_value, labels, created_at
		FROM metrics
		WHERE service_name = $1
		  AND metric_name = $2
		  AND timestamp > $3 
		ORDER BY timestamp DESC
		LIMIT 1000
	`
	// what is this timestamp for ? answer is that it is used to get the recent metrics in a duration
	// we ar ordering
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	//since := time.Now().Add(-duration) this is getting the time from duration means how, answer is it is getting the time from now and subtracting the duration from it
	since := time.Now().Add(-duration)                                    //we have added duration here because we are getting recent metrics in a duration
	rows, err := c.pool.Query(ctx, query, serviceName, metricName, since) // so this are getting the rows from the database on the basis of service name , metric name and since time
	if err != nil {
		return nil, fmt.Errorf("failed to query metrics: %w", err)
	}
	defer rows.Close()
	// means latest se purane ki taraf jaa rhe hain hum
	var metrics []*Metric
	for rows.Next() {
		var m Metric
		if err := rows.Scan(
			&m.ID,
			&m.Timestamp,
			&m.ServiceName,
			&m.MetricName,
			&m.MetricValue,
			&m.Labels,
			&m.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan metric row: %w", err)
		}
		metrics = append(metrics, &m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating metrics: %w", err)
	}

	return metrics, nil
}

func (c *PostgresClient) SaveDecision(ctx context.Context, decision *Decision) error {
	query := `
		INSERT INTO decisions (timestamp, pattern_detected, action_type, confidence, reason, parameters, executed)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := c.pool.QueryRow(
		ctx,
		query,
		decision.Timestamp,
		decision.PatternDetected,
		decision.ActionType,
		decision.Confidence,
		decision.Reason,
		decision.Parameters,
		decision.Executed,
	).Scan(&decision.ID, &decision.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to save decision: %w", err)
	}

	return nil
}

func (c *PostgresClient) SaveEvent(ctx context.Context, event *Event) error {
	query := `
		INSERT INTO events (timestamp, event_type, pod_name, namespace, message)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := c.pool.QueryRow(
		ctx,
		query,
		event.Timestamp,
		event.EventType,
		event.PodName,
		event.Namespace,
		event.Message,
	).Scan(&event.ID, &event.CreatedAt)

	if err != nil {
		c.logger.Error("Failed to save Kubernetes event",
			zap.Error(err),
			zap.String("event_type", event.EventType),
			zap.String("pod_name", event.PodName))
		return fmt.Errorf("failed to save event: %w", err)
	}

	c.logger.Debug("Saved Kubernetes event",
		zap.Int64("event_id", event.ID),
		zap.String("event_type", event.EventType),
		zap.String("pod_name", event.PodName),
		zap.String("namespace", event.Namespace))

	return nil
}

func (c *PostgresClient) BatchSaveMetrics(ctx context.Context, metrics []*Metric) error {
	if len(metrics) == 0 {
		c.logger.Debug("No metrics to save")
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Build rows for batch insert
	rows := make([][]any, 0, len(metrics))
	for _, metric := range metrics {
		rows = append(rows, []any{
			metric.Timestamp,
			metric.ServiceName,
			metric.MetricName,
			metric.MetricValue,
			metric.Labels,
		})
	}

	// Use PostgreSQL COPY command for efficient batch insert
	copyCount, err := c.pool.CopyFrom(
		ctx,
		pgx.Identifier{"metrics"},
		[]string{"timestamp", "service_name", "metric_name", "metric_value", "labels"},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		c.logger.Error("Failed to batch save metrics",
			zap.Error(err),
			zap.Int("attempted_count", len(metrics)))
		return fmt.Errorf("failed to copy metrics: %w", err)
	}

	c.logger.Info("Batch saved metrics to database",
		zap.Int64("saved_count", copyCount),
		zap.Int("metrics_count", len(metrics)))

	return nil
}

func (c *PostgresClient) GetPoolStats() *pgxpool.Stat {
	return c.pool.Stat()
}

func (c *PostgresClient) GetLatestMetric(
	ctx context.Context,
	serviceName string,
	metricName string,
) (*Metric, error) {
	query := `
		SELECT id, timestamp, service_name, metric_name, metric_value, labels, created_at
		FROM metrics
		WHERE service_name = $1
		  AND metric_name = $2
		ORDER BY timestamp DESC
		LIMIT 1
	`

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	//difference between get latest and get recent is that get latest is giving only one latest metric and get recent is giving multiple metrics in a duration
	var metric Metric
	err := c.pool.QueryRow(ctx, query, serviceName, metricName).Scan(
		&metric.ID,
		&metric.Timestamp,
		&metric.ServiceName,
		&metric.MetricName,
		&metric.MetricValue,
		&metric.Labels,
		&metric.CreatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get latest metric: %w", err)
	}

	return &metric, nil //kisi bhi metric ki latest value de raha hai ye function
}

func (c *PostgresClient) GetMetricStatistics(
	ctx context.Context,
	serviceName string,
	metricName string,
	duration time.Duration,
) (*MetricStats, error) {
	query := `
		SELECT 
			COUNT(*) as count,
			AVG(metric_value) as avg,
			MIN(metric_value) as min,
			MAX(metric_value) as max,
			STDDEV(metric_value) as stddev
		FROM metrics
		WHERE service_name = $1
		  AND metric_name = $2
		  AND timestamp > $3
	`

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	since := time.Now().Add(-duration)
	var stats MetricStats
	var stddev *float64

	err := c.pool.QueryRow(ctx, query, serviceName, metricName, since).Scan(
		&stats.Count,
		&stats.Avg,
		&stats.Min,
		&stats.Max,
		&stddev,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get metric statistics: %w", err)
	}

	if stddev != nil {
		stats.StdDev = *stddev
	}

	stats.ServiceName = serviceName
	stats.MetricName = metricName
	stats.Duration = duration

	return &stats, nil
}

func (c *PostgresClient) GetRecentEvents(
	ctx context.Context,
	namespace string,
	duration time.Duration,
) ([]*Event, error) {
	query := `
		SELECT id, timestamp, event_type, pod_name, namespace, message, created_at
		FROM events
		WHERE namespace = $1
		  AND timestamp > $2
		ORDER BY timestamp DESC
		LIMIT 100
	`

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	since := time.Now().Add(-duration)
	rows, err := c.pool.Query(ctx, query, namespace, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var events []*Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(
			&e.ID,
			&e.Timestamp,
			&e.EventType,
			&e.PodName,
			&e.Namespace,
			&e.Message,
			&e.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		events = append(events, &e)
	}

	return events, rows.Err()
}

func (c *PostgresClient) GetRecentDecisions(
	ctx context.Context,
	limit int,
) ([]*Decision, error) {
	query := `
		SELECT id, timestamp, pattern_detected, action_type, confidence, reason, parameters, executed, created_at
		FROM decisions
		ORDER BY timestamp DESC
		LIMIT $1
	`

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := c.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query decisions: %w", err)
	}
	defer rows.Close()

	var decisions []*Decision
	for rows.Next() {
		var d Decision
		if err := rows.Scan(
			&d.ID,
			&d.Timestamp,
			&d.PatternDetected,
			&d.ActionType,
			&d.Confidence,
			&d.Reason,
			&d.Parameters,
			&d.Executed,
			&d.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan decision: %w", err)
		}
		decisions = append(decisions, &d)
	}

	return decisions, rows.Err()
}

func (c *PostgresClient) GetDecisionStats(ctx context.Context, duration time.Duration) (*DecisionStats, error) {
	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE executed = true) as executed,
			COUNT(*) FILTER (WHERE executed = false) as pending,
			AVG(confidence) as avg_confidence
		FROM decisions
		WHERE timestamp > $1
	`

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	since := time.Now().Add(-duration)
	var stats DecisionStats

	err := c.pool.QueryRow(ctx, query, since).Scan(
		&stats.Total,
		&stats.Executed,
		&stats.Pending,
		&stats.AvgConfidence,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get decision stats: %w", err)
	}

	return &stats, nil
}

func (c *PostgresClient) DeleteOldMetrics(ctx context.Context, olderThan time.Duration) (int64, error) {
	query := `
		DELETE FROM metrics
		WHERE timestamp < $1
	`

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cutoff := time.Now().Add(-olderThan)
	result, err := c.pool.Exec(ctx, query, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old metrics: %w", err)
	}

	return result.RowsAffected(), nil
}

func (c *PostgresClient) GetAllServices(ctx context.Context) ([]string, error) {
	query := `
		SELECT DISTINCT service_name
		FROM metrics
		WHERE timestamp > $1
		ORDER BY service_name
	`

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	since := time.Now().Add(-24 * time.Hour)
	rows, err := c.pool.Query(ctx, query, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query services: %w", err)
	}
	defer rows.Close()

	var services []string
	for rows.Next() {
		var service string
		if err := rows.Scan(&service); err != nil {
			return nil, fmt.Errorf("failed to scan service: %w", err)
		}
		services = append(services, service)
	}

	return services, rows.Err()
}

func (c *PostgresClient) GetDecisionById(ctx context.Context, id string) (*Decision, error) {
	query := `
		SELECT id, timestamp, pattern_detected, action_type, confidence, reason, parameters, executed, created_at
		FROM decisions
		WHERE id = $1
	`

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var decision Decision
	err := c.pool.QueryRow(ctx, query, id).Scan(
		&decision.ID,
		&decision.Timestamp,
		&decision.PatternDetected,
		&decision.ActionType,
		&decision.Confidence,
		&decision.Reason,
		&decision.Parameters,
		&decision.Executed,
		&decision.CreatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("decision not found")
		}
		return nil, fmt.Errorf("failed to get decision: %w", err)
	}

	return &decision, nil
}

func (c *PostgresClient) GetPodEvents(ctx context.Context, podName string, duration time.Duration) ([]*Event, error) {
	query := `
		SELECT id, timestamp, event_type, pod_name, namespace, message, created_at
		FROM events
		WHERE pod_name = $1
		  AND timestamp > $2
		ORDER BY timestamp DESC
		LIMIT 100
	`

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	since := time.Now().Add(-duration)
	rows, err := c.pool.Query(ctx, query, podName, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query pod events: %w", err)
	}
	defer rows.Close()

	var events []*Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(
			&e.ID,
			&e.Timestamp,
			&e.EventType,
			&e.PodName,
			&e.Namespace,
			&e.Message,
			&e.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		events = append(events, &e)
	}

	return events, rows.Err()
}

/*
| SELECT variation       | meaning            |
| ---------------------- | ------------------ |
| SELECT columns         | show exact data    |
| SELECT COUNT(*)        | show how many rows |
| SELECT SUM(x)          | add all values     |
| SELECT AVG(x)          | average            |
| SELECT MIN(x)          | smallest           |
| SELECT MAX(x)          | largest            |
| SELECT DISTINCT column | remove duplicates  |
*/

// SELECT * FROM users;
// SELECT id, name FROM users;
// SELECT COUNT(*) FROM users;
// SELECT SUM(price) FROM orders;
// SELECT AVG(age) FROM persons;
// SELECT MAX(salary) FROM employees;
// SELECT MIN(salary) FROM employees;
// SELECT DISTINCT country FROM customers;
// SELECT * FROM users WHERE age > 18;
// SELECT * FROM orders ORDER BY date DESC;
// SELECT * FROM logs ORDER BY time DESC LIMIT 10;
