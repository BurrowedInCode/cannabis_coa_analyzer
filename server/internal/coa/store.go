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
	StoreCOAAnalysis(ctx context.Context, a *Analysis) error
	GetAllCOAAnalyses(ctx context.Context, limit int, offset int) ([]*AnalysisSummary, error)
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

func (s *Store) StoreCOAAnalysis(ctx context.Context, a *Analysis) error {
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
	err = tx.QueryRow(ctx, "INSERT INTO analyses (sample_name, seed_to_sale_number, sample_matrix, test_date, overall_pass, laboratory_id) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id", a.SampleName, a.SeedToSaleNumber, a.SampleMatrix, a.TestDate, a.OverallPass, labID).Scan(&analysisID)
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
