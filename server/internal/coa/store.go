package coa

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	db *pgxpool.Pool
}

type COAStore interface {
	StoreCOAAnalysis(ctx context.Context, a *Analysis, u Usage) error
	GetAllCOAAnalyses(ctx context.Context, limit int, offset int) ([]*AnalysisSummary, error)
	GetCOAAnalysis(ctx context.Context, id string) (*Analysis, error)
	UpdateCOAAnalysis(ctx context.Context, id string, a *Analysis) error
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

func (s *Store) StoreCOAAnalysis(ctx context.Context, a *Analysis, u Usage) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var labID string
	err = tx.QueryRow(ctx, "INSERT INTO laboratories (name, address, phone, certification) VALUES ($1, $2, $3, $4) RETURNING id", a.Laboratory.Name, a.Laboratory.Address, a.Laboratory.Phone, a.Laboratory.Certification).Scan(&labID)
	if err != nil {
		return fmt.Errorf("failed to insert laboratory: %w", err)
	}

	var analysisID string
	err = tx.QueryRow(ctx, "INSERT INTO analyses (sample_name, seed_to_sale_number, sample_matrix, test_date, overall_pass, laboratory_id, input_tokens, output_tokens) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id", a.SampleName, a.SeedToSaleNumber, a.SampleMatrix, a.TestDate, a.OverallPass, labID, u.InputTokens, u.OutputTokens).Scan(&analysisID)
	if err != nil {
		return fmt.Errorf("failed to insert analysis: %w", err)
	}

	for _, c := range a.Cannabinoids {
		_, err = tx.Exec(ctx,
			"INSERT INTO cannabinoids (name, value, unit, analysis_id) VALUES ($1, $2, $3, $4)",
			c.Name, c.Value, c.Unit, analysisID,
		)
		if err != nil {
			return fmt.Errorf("failed to insert cannabinoid: %w", err)
		}
	}

	for _, t := range a.Terpenes {
		_, err = tx.Exec(ctx,
			"INSERT INTO terpenes (name, value, unit, analysis_id) VALUES ($1, $2, $3, $4)",
			t.Name, t.Value, t.Unit, analysisID,
		)
		if err != nil {
			return fmt.Errorf("failed to insert terpene: %w", err)
		}
	}

	for _, ts := range a.Summary {
		_, err = tx.Exec(ctx,
			"INSERT INTO test_summaries (name, status, analysis_id) VALUES ($1, $2, $3)",
			ts.Name, ts.Status, analysisID,
		)
		if err != nil {
			return fmt.Errorf("failed to insert test summary: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (s *Store) GetAllCOAAnalyses(ctx context.Context, limit int, offset int) ([]*AnalysisSummary, error) {
	rows, err := s.db.Query(ctx, "SELECT id, sample_name, seed_to_sale_number, test_date, overall_pass FROM analyses LIMIT $1 OFFSET $2", limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query analyses: %w", err)
	}
	defer rows.Close()

	analysesSummary := []*AnalysisSummary{}

	for rows.Next() {
		var analysisSummary AnalysisSummary

		if err := rows.Scan(&analysisSummary.ID, &analysisSummary.SampleName, &analysisSummary.SeedToSaleNumber, &analysisSummary.TestDate, &analysisSummary.OverallPass); err != nil {
			return nil, fmt.Errorf("failed to scan rows: %w", err)
		}

		analysesSummary = append(analysesSummary, &analysisSummary)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate rows: %w", err)
	}

	return analysesSummary, nil
}

func (s *Store) GetCOAAnalysis(ctx context.Context, id string) (*Analysis, error) {
	analysis := &Analysis{}
	var analysisID string

	err := s.db.QueryRow(ctx, `
		SELECT a.id, a.sample_name, a.seed_to_sale_number, a.sample_matrix, a.test_date,
			a.overall_pass, l.name, l.address, l.phone, l.certification
		FROM analyses a
		INNER JOIN laboratories l ON a.laboratory_id = l.id
		WHERE a.id = $1`, id).Scan(
		&analysisID,
		&analysis.SampleName,
		&analysis.SeedToSaleNumber,
		&analysis.SampleMatrix,
		&analysis.TestDate,
		&analysis.OverallPass,
		&analysis.Laboratory.Name,
		&analysis.Laboratory.Address,
		&analysis.Laboratory.Phone,
		&analysis.Laboratory.Certification,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query analysis: %w", err)
	}

	cannabinoidRows, err := s.db.Query(ctx, "SELECT name, value, unit FROM cannabinoids WHERE analysis_id=$1", analysisID)
	if err != nil {
		return nil, fmt.Errorf("failed to query cannabinoids: %w", err)
	}
	defer cannabinoidRows.Close()

	for cannabinoidRows.Next() {
		var c Cannabinoid
		if err := cannabinoidRows.Scan(&c.Name, &c.Value, &c.Unit); err != nil {
			return nil, fmt.Errorf("failed to scan cannabinoid: %w", err)
		}
		analysis.Cannabinoids = append(analysis.Cannabinoids, c)
	}
	if err := cannabinoidRows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate cannabinoids: %w", err)
	}

	terpeneRows, err := s.db.Query(ctx, "SELECT name, value, unit FROM terpenes WHERE analysis_id=$1", analysisID)
	if err != nil {
		return nil, fmt.Errorf("failed to query terpenes: %w", err)
	}
	defer terpeneRows.Close()

	for terpeneRows.Next() {
		var t Terpene
		if err := terpeneRows.Scan(&t.Name, &t.Value, &t.Unit); err != nil {
			return nil, fmt.Errorf("failed to scan terpene: %w", err)
		}
		analysis.Terpenes = append(analysis.Terpenes, t)
	}
	if err := terpeneRows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate terpenes: %w", err)
	}

	summaryRows, err := s.db.Query(ctx, "SELECT name, status FROM test_summaries WHERE analysis_id=$1", analysisID)
	if err != nil {
		return nil, fmt.Errorf("failed to query test summaries: %w", err)
	}
	defer summaryRows.Close()

	for summaryRows.Next() {
		var ts TestSummary
		if err := summaryRows.Scan(&ts.Name, &ts.Status); err != nil {
			return nil, fmt.Errorf("failed to scan test summary: %w", err)
		}
		analysis.Summary = append(analysis.Summary, ts)
	}
	if err := summaryRows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate test summaries: %w", err)
	}

	return analysis, nil
}

func (s *Store) UpdateCOAAnalysis(ctx context.Context, id string, a *Analysis) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`UPDATE analyses
		 SET sample_name=$1, seed_to_sale_number=$2, sample_matrix=$3, test_date=$4, overall_pass=$5
		 WHERE id=$6`,
		a.SampleName, a.SeedToSaleNumber, a.SampleMatrix, a.TestDate, a.OverallPass, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update analysis: %w", err)
	}

	_, err = tx.Exec(ctx,
		`UPDATE laboratories
		 SET name=$1, address=$2, phone=$3, certification=$4
		 WHERE id=(SELECT laboratory_id FROM analyses WHERE id=$5)`,
		a.Laboratory.Name, a.Laboratory.Address, a.Laboratory.Phone, a.Laboratory.Certification, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update laboratory: %w", err)
	}

	if _, err = tx.Exec(ctx, "DELETE FROM cannabinoids WHERE analysis_id=$1", id); err != nil {
		return fmt.Errorf("failed to delete cannabinoids: %w", err)
	}
	for _, c := range a.Cannabinoids {
		_, err = tx.Exec(ctx,
			"INSERT INTO cannabinoids (name, value, unit, analysis_id) VALUES ($1, $2, $3, $4)",
			c.Name, c.Value, c.Unit, id,
		)
		if err != nil {
			return fmt.Errorf("failed to insert cannabinoid: %w", err)
		}
	}

	if _, err = tx.Exec(ctx, "DELETE FROM terpenes WHERE analysis_id=$1", id); err != nil {
		return fmt.Errorf("failed to delete terpenes: %w", err)
	}
	for _, t := range a.Terpenes {
		_, err = tx.Exec(ctx,
			"INSERT INTO terpenes (name, value, unit, analysis_id) VALUES ($1, $2, $3, $4)",
			t.Name, t.Value, t.Unit, id,
		)
		if err != nil {
			return fmt.Errorf("failed to insert terpene: %w", err)
		}
	}

	if _, err = tx.Exec(ctx, "DELETE FROM test_summaries WHERE analysis_id=$1", id); err != nil {
		return fmt.Errorf("failed to delete test summaries: %w", err)
	}
	for _, ts := range a.Summary {
		_, err = tx.Exec(ctx,
			"INSERT INTO test_summaries (name, status, analysis_id) VALUES ($1, $2, $3)",
			ts.Name, ts.Status, id,
		)
		if err != nil {
			return fmt.Errorf("failed to insert test summary: %w", err)
		}
	}

	return tx.Commit(ctx)
}
