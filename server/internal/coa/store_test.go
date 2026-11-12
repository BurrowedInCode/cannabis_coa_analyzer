//go:build integration

package coa

import (
	"cmp"
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()

	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Fatal("TEST_DATABASE_URL must be set for integration tests")
	}

	schemaName := "coa_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	quotedSchema := pgx.Identifier{schemaName}.Sanitize()

	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		t.Fatalf("parsing config: %v", err)
	}

	ctx := context.Background()
	config.ConnConfig.RuntimeParams["search_path"] = quotedSchema

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatalf("creating connection pool: %v", err)
	}
	t.Cleanup(func() {
		pool.Close()
	})

	_, err = pool.Exec(ctx, "CREATE SCHEMA "+quotedSchema)
	if err != nil {
		t.Fatalf("creating schema: %v", err)
	}
	t.Cleanup(func() {
		_, err := pool.Exec(ctx, "DROP SCHEMA "+quotedSchema+" CASCADE")
		if err != nil {
			t.Errorf("dropping schema: %v", err)
		}
	})

	migrationFiles, err := filepath.Glob("../../db/migrations/*.up.sql")
	if err != nil {
		t.Fatalf("finding migrations: %v", err)
	}

	if len(migrationFiles) == 0 {
		t.Fatal("no migration files found")
	}

	for _, f := range migrationFiles {
		sql, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("reading %s: %v ", f, err)
		}
		_, err = pool.Exec(ctx, string(sql))
		if err != nil {
			t.Fatalf("applying migration %s: %v", f, err)
		}
	}
	return NewStore(pool)
}

func TestStoreCOAAnalysisRoundTrip(t *testing.T) {
	want := &Analysis{
		Laboratory: Laboratory{
			Name:          "example lab",
			Address:       "123 main st",
			Phone:         "123-456-7890",
			Certification: "cert-12345",
		},
		SampleName:       "sample name",
		SeedToSaleNumber: "1234567890",
		SampleMatrix:     "Flower",
		TestDate:         time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC),
		OverallPass:      true,
		Cannabinoids: []Cannabinoid{
			{Name: "THC", Value: 23, Unit: "%"},
			{Name: "CBD", Value: 10, Unit: "%"},
		},
		Terpenes: []Terpene{
			{Name: "Pinene", Value: 1.5, Unit: "%"},
		},
		Summary: []TestSummary{
			{Name: "Pesticides", Status: "pass"},
			{Name: "Heavy Metals", Status: "pass"},
		},
	}

	ctx := context.Background()
	store := newTestStore(t)

	usage := Usage{InputTokens: 120, OutputTokens: 45}

	err := store.StoreCOAAnalysis(ctx, want, usage)
	if err != nil {
		t.Fatalf("store analysis: %v", err)
	}

	var id string
	var gotUsage Usage

	err = store.db.QueryRow(ctx, "SELECT id, input_tokens, output_tokens FROM analyses").Scan(&id, &gotUsage.InputTokens, &gotUsage.OutputTokens)
	if err != nil {
		t.Fatalf("query row: %v", err)
	}

	if gotUsage.InputTokens != usage.InputTokens {
		t.Errorf("input tokens = %d, want %d", gotUsage.InputTokens, usage.InputTokens)
	}

	if gotUsage.OutputTokens != usage.OutputTokens {
		t.Errorf("output tokens = %d, want %d", gotUsage.OutputTokens, usage.OutputTokens)
	}

	got, err := store.GetCOAAnalysis(ctx, id)
	if err != nil {
		t.Fatalf("get analysis: %v", err)
	}

	if got.SampleName != want.SampleName {
		t.Errorf("sample name = %q, want %q", got.SampleName, want.SampleName)
	}

	if got.SeedToSaleNumber != want.SeedToSaleNumber {
		t.Errorf("seed to sale number = %q, want %q", got.SeedToSaleNumber, want.SeedToSaleNumber)
	}

	if got.SampleMatrix != want.SampleMatrix {
		t.Errorf("sample matrix = %q, want %q", got.SampleMatrix, want.SampleMatrix)
	}

	if got.OverallPass != want.OverallPass {
		t.Errorf("overall pass = %t, want %t", got.OverallPass, want.OverallPass)
	}

	if !got.TestDate.Equal(want.TestDate) {
		t.Errorf("test date = %v, want %v", got.TestDate, want.TestDate)
	}

	if got.Laboratory != want.Laboratory {
		t.Errorf("laboratory = %+v, want %+v", got.Laboratory, want.Laboratory)
	}

	byCannabinoidName := func(a, b Cannabinoid) int {
		return cmp.Compare(a.Name, b.Name)
	}

	slices.SortFunc(got.Cannabinoids, byCannabinoidName)
	slices.SortFunc(want.Cannabinoids, byCannabinoidName)

	if !slices.Equal(got.Cannabinoids, want.Cannabinoids) {
		t.Errorf("cannabinoids = got %+v, want %+v", got.Cannabinoids, want.Cannabinoids)
	}

	byTerpeneName := func(a, b Terpene) int {
		return cmp.Compare(a.Name, b.Name)
	}

	slices.SortFunc(got.Terpenes, byTerpeneName)
	slices.SortFunc(want.Terpenes, byTerpeneName)

	if !slices.Equal(got.Terpenes, want.Terpenes) {
		t.Errorf("terpenes = got %+v, want %+v", got.Terpenes, want.Terpenes)
	}

	bySummaryName := func(a, b TestSummary) int {
		return cmp.Compare(a.Name, b.Name)
	}

	slices.SortFunc(got.Summary, bySummaryName)
	slices.SortFunc(want.Summary, bySummaryName)

	if !slices.Equal(got.Summary, want.Summary) {
		t.Errorf("summaries = got %+v, want %+v", got.Summary, want.Summary)
	}
}

func TestStoreCOAAnalysisRollsBackOnInsertError(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	_, err := store.db.Exec(ctx, `
		ALTER TABLE test_summaries
		ADD CONSTRAINT test_reject_summary
		CHECK (status <> 'force_failure')
		`)
	if err != nil {
		t.Fatalf("adding test constraint: %v", err)
	}
	analysis := &Analysis{
		Laboratory: Laboratory{
			Name:          "example lab",
			Address:       "123 main st",
			Phone:         "123-456-7890",
			Certification: "cert-12345",
		},
		SampleName:       "sample name",
		SeedToSaleNumber: "1234567890",
		SampleMatrix:     "Flower",
		TestDate:         time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC),
		OverallPass:      true,
		Cannabinoids: []Cannabinoid{
			{Name: "THC", Value: 23, Unit: "%"},
			{Name: "CBD", Value: 10, Unit: "%"},
		},
		Terpenes: []Terpene{
			{Name: "Pinene", Value: 1.5, Unit: "%"},
		},
		Summary: []TestSummary{
			{Name: "Pesticides", Status: "pass"},
			{Name: "Heavy Metals", Status: "force_failure"},
		},
	}

	err = store.StoreCOAAnalysis(ctx, analysis, Usage{
		InputTokens:  120,
		OutputTokens: 45,
	})
	if err == nil {
		t.Fatalf("store analysis succeeded, want a constraint violation")
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		t.Fatalf("error = %v, want a PostgreSQL error", err)
	}

	if pgErr.Code != "23514" || pgErr.ConstraintName != "test_reject_summary" {
		t.Fatalf("unexpected database error: %v", pgErr)
	}

	tables := []string{
		"laboratories",
		"analyses",
		"cannabinoids",
		"terpenes",
		"test_summaries",
	}

	for _, table := range tables {
		var count int
		err := store.db.QueryRow(
			ctx, "SELECT count(*) FROM "+pgx.Identifier{table}.Sanitize(),
		).Scan(&count)
		if err != nil {
			t.Fatalf("counting %s after rollback: %v", table, err)
		}

		if count != 0 {
			t.Errorf("%s rows after rollback = %d, want 0", table, count)
		}
	}
}

func TestGetAllCOAAnalyses(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	newer := &Analysis{
		Laboratory: Laboratory{
			Name:          "Example Lab",
			Address:       "123 Main St",
			Phone:         "555-0100",
			Certification: "CERT-001",
		},
		SampleName:       "Blue Dream",
		SeedToSaleNumber: "SALE-001",
		SampleMatrix:     "Flower",
		TestDate:         time.Date(2026, time.January, 2, 12, 0, 0, 0, time.UTC),
		OverallPass:      true,
	}

	older := &Analysis{
		Laboratory: Laboratory{
			Name:          "Another Lab",
			Address:       "456 Oak St",
			Phone:         "555-0200",
			Certification: "CERT-002",
		},
		SampleName:       "Sour Diesel",
		SeedToSaleNumber: "SALE-002",
		SampleMatrix:     "Flower",
		TestDate:         time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC),
		OverallPass:      false,
	}

	usage := Usage{InputTokens: 120, OutputTokens: 45}

	err := store.StoreCOAAnalysis(ctx, newer, usage)
	if err != nil {
		t.Fatalf("store analysis: %v", err)
	}

	err = store.StoreCOAAnalysis(ctx, older, usage)
	if err != nil {
		t.Fatalf("store analysis: %v", err)
	}

	var newerID, olderID string
	err = store.db.QueryRow(ctx,
		"SELECT id FROM analyses WHERE seed_to_sale_number = $1", newer.SeedToSaleNumber,
	).Scan(&newerID)
	if err != nil {
		t.Fatalf("query newer analysis ID: %v", err)
	}

	err = store.db.QueryRow(ctx,
		"SELECT id FROM analyses WHERE seed_to_sale_number = $1", older.SeedToSaleNumber,
	).Scan(&olderID)
	if err != nil {
		t.Fatalf("query older analysis ID: %v", err)
	}

	got, err := store.GetAllCOAAnalyses(ctx, 5, 0)
	if err != nil {
		t.Fatalf("get all analyses: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("analyses count = %d, want 2", len(got))
	}

	if got[0].ID != newerID {
		t.Errorf("first analysis ID = %q, want %q", got[0].ID, newerID)
	}

	if got[0].SampleName != newer.SampleName {
		t.Errorf("first sample name = %q, want %q", got[0].SampleName, newer.SampleName)
	}

	if got[0].SeedToSaleNumber != newer.SeedToSaleNumber {
		t.Errorf("first seed to sale number = %q, want %q", got[0].SeedToSaleNumber, newer.SeedToSaleNumber)
	}

	if got[0].OverallPass != newer.OverallPass {
		t.Errorf("first overall pass = %t, want %t", got[0].OverallPass, newer.OverallPass)
	}

	if !got[0].TestDate.Equal(newer.TestDate) {
		t.Errorf("first test date = %v, want %v", got[0].TestDate, newer.TestDate)
	}

	if got[1].ID != olderID {
		t.Errorf("second analysis ID = %q, want %q", got[1].ID, olderID)
	}

	if got[1].SampleName != older.SampleName {
		t.Errorf("second sample name = %q, want %q", got[1].SampleName, older.SampleName)
	}

	if got[1].SeedToSaleNumber != older.SeedToSaleNumber {
		t.Errorf("second seed to sale number = %q, want %q", got[1].SeedToSaleNumber, older.SeedToSaleNumber)
	}

	if got[1].OverallPass != older.OverallPass {
		t.Errorf("second overall pass = %t, want %t", got[1].OverallPass, older.OverallPass)
	}

	if !got[1].TestDate.Equal(older.TestDate) {
		t.Errorf("first test date = %v, want %v", got[1].TestDate, older.TestDate)
	}
}

func TestGetAllCOAAnalysesEmpty(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	got, err := store.GetAllCOAAnalyses(ctx, 5, 0)
	if err != nil {
		t.Fatalf("getting all analyses: %v", err)
	}

	if len(got) != 0 {
		t.Errorf("analyses count = %d, want 0", len(got))
	}
}

func TestGetAllCOAAnalysesLimit(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	first := &Analysis{
		Laboratory: Laboratory{
			Name:          "Example Lab",
			Address:       "123 Main St",
			Phone:         "555-0100",
			Certification: "CERT-001",
		},
		SampleName:       "Blue Dream",
		SeedToSaleNumber: "SALE-001",
		SampleMatrix:     "Flower",
		TestDate:         time.Date(2026, time.January, 2, 12, 0, 0, 0, time.UTC),
		OverallPass:      true,
	}

	second := &Analysis{
		Laboratory: Laboratory{
			Name:          "Another Lab",
			Address:       "456 Oak St",
			Phone:         "555-0200",
			Certification: "CERT-002",
		},
		SampleName:       "Sour Diesel",
		SeedToSaleNumber: "SALE-002",
		SampleMatrix:     "Flower",
		TestDate:         time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC),
		OverallPass:      false,
	}

	usage := Usage{InputTokens: 120, OutputTokens: 45}
	err := store.StoreCOAAnalysis(ctx, first, usage)
	if err != nil {
		t.Fatalf("storing first analysis: %v", err)
	}

	err = store.StoreCOAAnalysis(ctx, second, usage)
	if err != nil {
		t.Fatalf("storing second analysis: %v", err)
	}

	got, err := store.GetAllCOAAnalyses(ctx, 1, 0)
	if err != nil {
		t.Fatalf("getting all analyses: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("analyses count = %d, expected 1", len(got))
	}

	if got[0].SampleName != "Blue Dream" {
		t.Errorf("got analysis sample name %s, wanted 'Blue Dream'", got[0].SampleName)
	}
}

func TestGetAllCOAAnalysesOffset(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	first := &Analysis{
		Laboratory: Laboratory{
			Name:          "Example Lab",
			Address:       "123 Main St",
			Phone:         "555-0100",
			Certification: "CERT-001",
		},
		SampleName:       "Blue Dream",
		SeedToSaleNumber: "SALE-001",
		SampleMatrix:     "Flower",
		TestDate:         time.Date(2026, time.January, 2, 12, 0, 0, 0, time.UTC),
		OverallPass:      true,
	}

	second := &Analysis{
		Laboratory: Laboratory{
			Name:          "Another Lab",
			Address:       "456 Oak St",
			Phone:         "555-0200",
			Certification: "CERT-002",
		},
		SampleName:       "Sour Diesel",
		SeedToSaleNumber: "SALE-002",
		SampleMatrix:     "Flower",
		TestDate:         time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC),
		OverallPass:      false,
	}

	usage := Usage{InputTokens: 120, OutputTokens: 45}

	err := store.StoreCOAAnalysis(ctx, first, usage)
	if err != nil {
		t.Fatalf("storing first analysis: %v", err)
	}

	err = store.StoreCOAAnalysis(ctx, second, usage)
	if err != nil {
		t.Fatalf("storing second analysis: %v", err)
	}

	t.Run("within results", func(t *testing.T) {
		got, err := store.GetAllCOAAnalyses(ctx, 1, 1)
		if err != nil {
			t.Fatalf("getting all analyses: %v", err)
		}

		if len(got) != 1 {
			t.Fatalf("analyses count = %d, expected 1", len(got))
		}

		if got[0].SampleName != "Sour Diesel" {
			t.Errorf("got analysis sample name %s, wanted 'Sour Diesel'", got[0].SampleName)
		}
	})

	t.Run("beyond results", func(t *testing.T) {

		got, err := store.GetAllCOAAnalyses(ctx, 1, 3)
		if err != nil {
			t.Fatalf("getting all analyses: %v", err)
		}

		if len(got) != 0 {
			t.Errorf("analyses count = %d, expected 0", len(got))
		}
	})
}

func TestGetAllCOAAnalysesDateTieBreak(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	first := &Analysis{
		Laboratory: Laboratory{
			Name:          "Example Lab",
			Address:       "123 Main St",
			Phone:         "555-0100",
			Certification: "CERT-001",
		},
		SampleName:       "Blue Dream",
		SeedToSaleNumber: "SALE-001",
		SampleMatrix:     "Flower",
		TestDate:         time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC),
		OverallPass:      true,
	}

	second := &Analysis{
		Laboratory: Laboratory{
			Name:          "Another Lab",
			Address:       "456 Oak St",
			Phone:         "555-0200",
			Certification: "CERT-002",
		},
		SampleName:       "Sour Diesel",
		SeedToSaleNumber: "SALE-002",
		SampleMatrix:     "Flower",
		TestDate:         time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC),
		OverallPass:      false,
	}

	usage := Usage{InputTokens: 120, OutputTokens: 45}

	err := store.StoreCOAAnalysis(ctx, first, usage)
	if err != nil {
		t.Fatalf("storing first analysis: %v", err)
	}

	err = store.StoreCOAAnalysis(ctx, second, usage)
	if err != nil {
		t.Fatalf("storing second analysis: %v", err)
	}

	got, err := store.GetAllCOAAnalyses(ctx, 2, 0)
	if err != nil {
		t.Fatalf("getting all analyses: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("analyses count = %d, want 2", len(got))
	}

	if got[0].ID <= got[1].ID {
		t.Errorf("IDs = %q, %q, want descending order", got[0].ID, got[1].ID)
	}
}

func TestGetAllCOAAnalysesDatabaseErrors(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	_, err := store.db.Exec(ctx, "ALTER TABLE analyses RENAME TO unavailable_analyses")
	if err != nil {
		t.Fatalf("rename analyses table: %v", err)
	}

	got, err := store.GetAllCOAAnalyses(ctx, 1, 0)
	if err == nil {
		t.Fatalf("getting all analyses succeeded, want an undefined-table error")
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		t.Fatalf("error = %v, want a PostgreSQL error", err)
	}

	if pgErr.Code != "42P01" {
		t.Errorf("expected undefined table, got %s", pgErr.Code)
	}

	if got != nil {
		t.Errorf("analyses = %v, want nil", got)
	}
}
