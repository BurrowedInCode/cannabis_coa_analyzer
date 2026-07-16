package eval

import (
	"math"
	"strings"
	"time"

	"github.com/BurrowedInCode/cannabis_coa_analyzer/internal/coa"
)

// Score grades one extraction against its hand-written answer key (expected).
// It returns how many fields matched (correct) out of how many were checked
// (total), plus the names of the fields that missed so you can see what to fix.
func Score(expected, actual *coa.Analysis) (correct, total int, missed []string) {
	// --- scalar fields: one tick each ---
	if norm(expected.Laboratory.Name) == norm(actual.Laboratory.Name) {
		correct++
	} else {
		missed = append(missed, "laboratory.name")
	}
	total++

	if norm(expected.Laboratory.Address) == norm(actual.Laboratory.Address) {
		correct++
	} else {
		missed = append(missed, "laboratory.address")
	}
	total++

	if norm(expected.Laboratory.Phone) == norm(actual.Laboratory.Phone) {
		correct++
	} else {
		missed = append(missed, "laboratory.phone")
	}
	total++

	if norm(expected.Laboratory.Certification) == norm(actual.Laboratory.Certification) {
		correct++
	} else {
		missed = append(missed, "laboratory.certification")
	}
	total++

	if norm(expected.SampleName) == norm(actual.SampleName) {
		correct++
	} else {
		missed = append(missed, "sample_name")
	}
	total++

	if norm(expected.SeedToSaleNumber) == norm(actual.SeedToSaleNumber) {
		correct++
	} else {
		missed = append(missed, "seed_to_sale_number")
	}
	total++

	if norm(expected.SampleMatrix) == norm(actual.SampleMatrix) {
		correct++
	} else {
		missed = append(missed, "sample_matrix")
	}
	total++

	if sameDate(expected.TestDate, actual.TestDate) {
		correct++
	} else {
		missed = append(missed, "test_date")
	}
	total++

	if expected.OverallPass == actual.OverallPass {
		correct++
	} else {
		missed = append(missed, "overall_pass")
	}
	total++

	// --- list fields: search for each expected entry ---
	for _, want := range expected.Cannabinoids {
		found := false
		for _, got := range actual.Cannabinoids {
			if norm(want.Name) == norm(got.Name) && floatEqual(want.Value, got.Value) {
				found = true
				break
			}
		}
		if found {
			correct++
		} else {
			missed = append(missed, "cannabinoid: "+want.Name)
		}
		total++
	}

	for _, want := range expected.Terpenes {
		found := false
		for _, got := range actual.Terpenes {
			if norm(want.Name) == norm(got.Name) && floatEqual(want.Value, got.Value) {
				found = true
				break
			}
		}
		if found {
			correct++
		} else {
			missed = append(missed, "terpene: "+want.Name)
		}
		total++
	}

	for _, want := range expected.Summary {
		found := false
		for _, got := range actual.Summary {
			if norm(want.Name) == norm(got.Name) && norm(want.Status) == norm(got.Status) {
				found = true
				break
			}
		}
		if found {
			correct++
		} else {
			missed = append(missed, "summary: "+want.Name)
		}
		total++
	}

	return correct, total, missed
}

// norm makes string comparison forgiving: "Green Labs" and "green labs " match.
func norm(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}

// sameDate compares by calendar day, ignoring time-of-day and timezone.
func sameDate(a, b time.Time) bool {
	return a.Format("2006-01-02") == b.Format("2006-01-02")
}

// floatEqual treats two measured numbers as equal within a small tolerance, so
// rounding (25.34 vs 25.3) doesn't count as wrong. Loosen if the model is right
// but flagged.
func floatEqual(want, got float64) bool {
	return math.Abs(want-got) <= 0.02*math.Abs(want)+0.01
}
