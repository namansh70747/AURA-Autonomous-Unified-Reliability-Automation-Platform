package storage

import (
	"context"
	"encoding/json"
	"time"

	"github.com/namansh70747/AURA-Autonomous-Unified-Reliability-Automation-Platform/pkg/logger"
	"go.uber.org/zap"
)

type DiagnosisRecord struct {
	ID             int64                  `db:"id"`
	ServiceName    string                 `db:"service_name"`
	ProblemType    string                 `db:"problem_type"`
	Confidence     float64                `db:"confidence"`
	Severity       string                 `db:"severity"`
	Evidence       map[string]interface{} `db:"evidence"`
	Recommendation string                 `db:"recommendation"`
	Timestamp      time.Time              `db:"timestamp"`
}

func (p *PostgresClient) SaveDiagnosis(ctx context.Context, diagnosis *DiagnosisRecord) error {
	evidenceJSON, err := json.Marshal(diagnosis.Evidence)
	if err != nil {
		logger.Error("Failed to marshal evidence",
			zap.String("service", diagnosis.ServiceName),
			zap.Error(err),
		)
		return err
	}

	query := `
        INSERT INTO diagnoses (
            service_name, problem_type, confidence, severity, 
            evidence, recommendation, timestamp
        )
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        RETURNING id
    `

	var id int64
	err = p.pool.QueryRow(
		ctx,
		query,
		diagnosis.ServiceName,
		diagnosis.ProblemType,
		diagnosis.Confidence,
		diagnosis.Severity,
		evidenceJSON,
		diagnosis.Recommendation,
		diagnosis.Timestamp,
	).Scan(&id)

	if err != nil {
		logger.Error("Failed to save diagnosis",
			zap.String("service", diagnosis.ServiceName),
			zap.Error(err),
		)
		return err
	}
	logger.Info("Diagnosis saved",
		zap.String("service", diagnosis.ServiceName),
		zap.Int64("id", id),
	)

	return nil
}

func (p *PostgresClient) GetRecentDiagnosis(ctx context.Context, serviceName string, limit int) ([]*DiagnosisRecord, error) {
	query := `
        SELECT id, service_name, problem_type, confidence, severity,
               evidence, recommendation, timestamp
        FROM diagnoses
        WHERE service_name = $1
        ORDER BY timestamp DESC
        LIMIT $2
    `

	rows, err := p.pool.Query(ctx, query, serviceName, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var diagnoses []*DiagnosisRecord

	for rows.Next() {
		var d DiagnosisRecord
		var evidenceJSON []byte

		err := rows.Scan(
			&d.ID,
			&d.ServiceName,
			&d.ProblemType,
			&d.Confidence,
			&d.Severity,
			&evidenceJSON,
			&d.Recommendation,
			&d.Timestamp,
		)

		if err != nil {
			logger.Error("Failed to scan diagnosis", zap.Error(err))
			continue
		}

		if err := json.Unmarshal(evidenceJSON, &d.Evidence); err != nil {
			logger.Error("Failed to unmarshal evidence", zap.Error(err))
			continue
		}

		diagnoses = append(diagnoses, &d)
	}
	return diagnoses, nil
}
