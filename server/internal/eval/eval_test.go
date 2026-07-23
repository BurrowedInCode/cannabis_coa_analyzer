package eval

import (
	"testing"
	"time"

	"github.com/BurrowedInCode/cannabis_coa_analyzer/internal/coa"
)

// fixture returns a fresh, fully-populated Analysis. Each call builds a new
// value, so a case can take one and mutate a single field without touching
// the others.
func fixture() *coa.Analysis {
	return &coa.Analysis{
		Laboratory: coa.Laboratory{
			Name:          "TerpLife Labs",
			Address:       "10350 Fisher Ave, Tampa, Florida 33619",
			Phone:         "813-726-3103",
			Certification: "CMTL-00010",
		},
		SampleName:       "DoSiDos",
		SeedToSaleNumber: "5458711155557228",
		SampleMatrix:     "Flower Inhalable",
		TestDate:         time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC),
		OverallPass:      true,
		Cannabinoids: []coa.Cannabinoid{
			{Name: "Total THC", Value: 29.3, Unit: "%"},
			{Name: "Total CBD", Value: 0.0621, Unit: "%"},
		},
		Terpenes: []coa.Terpene{
			{Name: "beta-Myrcene", Value: 1.04, Unit: "%"},
			{Name: "D-Limonene", Value: 0.224, Unit: "%"},
		},
		Summary: []coa.TestSummary{
			{Name: "Microbials", Status: "Pass"},
			{Name: "Pesticides", Status: "Pass"},
			{Name: "Heavy Metals", Status: "Pass"},
		},
	}
}

func wrongFixture() *coa.Analysis {
	return &coa.Analysis{
		Laboratory: coa.Laboratory{
			Name:          "Modern Canna",
			Address:       "4705 Old Rd 37, Lakeland, FL 33813",
			Phone:         "863-608-7800",
			Certification: "CMTL-0005",
		},
		SampleName:       "Golden Pineapple",
		SeedToSaleNumber: "9854463684273110",
		SampleMatrix:     "Whole Flower",
		TestDate:         time.Date(2026, 3, 28, 0, 0, 0, 0, time.UTC),
		OverallPass:      false,
		Cannabinoids: []coa.Cannabinoid{
			{Name: "Fake Total THC", Value: 30.1, Unit: "%"},
			{Name: "Fake Total CBD", Value: 0.05, Unit: "%"},
		},
		Terpenes: []coa.Terpene{
			{Name: "beta-Caryophyllene", Value: 1.01, Unit: "%"},
			{Name: "alpha-Pinene", Value: 0.22, Unit: "%"},
		},
		Summary: []coa.TestSummary{
			{Name: "Microbials", Status: "Fail"},
			{Name: "Pesticides", Status: "Fail"},
			{Name: "Heavy Metals", Status: "Fail"},
		},
	}
}

func wrongSummaryValue() *coa.Analysis {
	a := fixture()
	a.Summary = []coa.TestSummary{
		{Name: "Microbials", Status: "Pass"},
		{Name: "Pesticides", Status: "Pass"},
		{Name: "Heavy Metals", Status: "Fail"},
	}
	return a
}

func cannabinoidExtraDecimal() *coa.Analysis {
	a := fixture()
	a.Cannabinoids = []coa.Cannabinoid{
		{Name: "Total THC", Value: 29.32, Unit: "%"},
		{Name: "Total CBD", Value: 0.0621, Unit: "%"},
	}
	return a
}

func missingCannabinoid() *coa.Analysis {
	a := fixture()
	a.Cannabinoids = []coa.Cannabinoid{
		{Name: "Total CBD", Value: 0.0621, Unit: "%"},
	}
	return a
}

func TestScore(t *testing.T) {
	tests := []struct {
		name        string
		expected    *coa.Analysis
		actual      *coa.Analysis
		wantCorrect int
		wantTotal   int
	}{
		{
			name:        "identical scores full marks",
			expected:    fixture(),
			actual:      fixture(),
			wantCorrect: 16,
			wantTotal:   16,
		},
		{
			name:        "all-wrong",
			expected:    fixture(),
			actual:      wrongFixture(),
			wantCorrect: 0,
			wantTotal:   16,
		},
		{
			name:        "single test summary value wrong",
			expected:    fixture(),
			actual:      wrongSummaryValue(),
			wantCorrect: 15,
			wantTotal:   16,
		},
		{
			name:        "handles float tolerance",
			expected:    fixture(),
			actual:      cannabinoidExtraDecimal(),
			wantCorrect: 16,
			wantTotal:   16,
		},
		{
			name:        "missing cannabinoid",
			expected:    fixture(),
			actual:      missingCannabinoid(),
			wantCorrect: 15,
			wantTotal:   16,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			correct, total, _ := Score(tc.expected, tc.actual)
			if correct != tc.wantCorrect {
				t.Errorf("correct = %d, want %d", correct, tc.wantCorrect)
			}
			if total != tc.wantTotal {
				t.Errorf("total = %d, want %d", total, tc.wantTotal)
			}
		})
	}
}

func TestScoreMissingCannabinoidReportsNotFound(t *testing.T) {
	_, _, missed := Score(fixture(), missingCannabinoid())

	found := false
	for _, m := range missed {
		if m.Field == "cannabinoid: Total THC" {
			found = true
			if m.Got != "(not found)" {
				t.Errorf("Got = %q, want %q", m.Got, "(not found)")
			}
		}
	}
	if !found {
		t.Error("no miss recorded for the dropped cannabinoid")
	}
}
