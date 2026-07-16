package eval

import (
	"math"
	"strings"
	"time"

	"github.com/BurrowedInCode/cannabis_coa_analyzer/internal/coa"
)

// Score grades one extraction against its hand-written answer key (expected).
// It returns how many fields matched (correct) out of how many were checked
// (total). The caller divides to get an accuracy percentage.
func Score(expected, actual *coa.Analysis) (correct, total int) {
	// --- scalar fields: one tick each ---
	if norm(expected.Laboratory.Name) == norm(actual.Laboratory.Name) {
		correct++
	}
	total++

	if norm(expected.Laboratory.Address) == norm(actual.Laboratory.Address) {
		correct++
	}
	total++

	if norm(expected.Laboratory.Phone) == norm(actual.Laboratory.Phone) {
		correct++
	}
	total++

	if norm(expected.Laboratory.Certification) == norm(actual.Laboratory.Certification) {
		correct++
	}
	total++

	if norm(expected.SampleName) == norm(actual.SampleName) {
		correct++
	}
	total++

	if norm(expected.SeedToSaleNumber) == norm(actual.SeedToSaleNumber) {
		correct++
	}
	total++

	if norm(expected.SampleMatrix) == norm(actual.SampleMatrix) {
		correct++
	}
	total++

	if sameDate(expected.TestDate, actual.TestDate) {
		correct++
	}
	total++

	if expected.OverallPass == actual.OverallPass {
		correct++
	}
	total++

	// --- list fields: search for each expected entry ---
	for _, want := range expected.Cannabinoids {
		for _, got := range actual.Cannabinoids {
			if norm(want.Name) == norm(got.Name) && floatEqual(want.Value, got.Value) {
				correct++
				break
			}
		}
		total++
	}

	for _, want := range expected.Terpenes {
		for _, got := range actual.Terpenes {
			if norm(want.Name) == norm(got.Name) && floatEqual(want.Value, got.Value) {
				correct++
				break
			}
		}
		total++
	}

	for _, want := range expected.Summary {
		for _, got := range actual.Summary {
			if norm(want.Name) == norm(got.Name) && norm(want.Status) == norm(got.Status) {
				correct++
				break
			}
		}
		total++
	}

	return correct, total
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
