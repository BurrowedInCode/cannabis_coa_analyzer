package eval

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/BurrowedInCode/cannabis_coa_analyzer/internal/coa"
)

type Miss struct {
	Field    string
	Expected string
	Got      string
}

// Score grades one extraction against its hand-written answer key (expected).
// It returns how many fields matched (correct) out of how many were checked
// (total), plus a Miss for each field that didn't match so you can see what to fix.
func Score(expected, actual *coa.Analysis) (correct, total int, missed []Miss) {
	// --- scalar fields: one tick each ---
	if norm(expected.Laboratory.Name) == norm(actual.Laboratory.Name) {
		correct++
	} else {
		missed = append(missed, Miss{"laboratory.name", expected.Laboratory.Name, actual.Laboratory.Name})
	}
	total++

	if norm(expected.Laboratory.Address) == norm(actual.Laboratory.Address) {
		correct++
	} else {
		missed = append(missed, Miss{"laboratory.address", expected.Laboratory.Address, actual.Laboratory.Address})
	}
	total++

	if norm(expected.Laboratory.Phone) == norm(actual.Laboratory.Phone) {
		correct++
	} else {
		missed = append(missed, Miss{"laboratory.phone", expected.Laboratory.Phone, actual.Laboratory.Phone})
	}
	total++

	if norm(expected.Laboratory.Certification) == norm(actual.Laboratory.Certification) {
		correct++
	} else {
		missed = append(missed, Miss{"laboratory.certification", expected.Laboratory.Certification, actual.Laboratory.Certification})
	}
	total++

	if norm(expected.SampleName) == norm(actual.SampleName) {
		correct++
	} else {
		missed = append(missed, Miss{"sample_name", expected.SampleName, actual.SampleName})
	}
	total++

	if norm(expected.SeedToSaleNumber) == norm(actual.SeedToSaleNumber) {
		correct++
	} else {
		missed = append(missed, Miss{"seed_to_sale_number", expected.SeedToSaleNumber, actual.SeedToSaleNumber})
	}
	total++

	if norm(expected.SampleMatrix) == norm(actual.SampleMatrix) {
		correct++
	} else {
		missed = append(missed, Miss{"sample_matrix", expected.SampleMatrix, actual.SampleMatrix})
	}
	total++

	if sameDate(expected.TestDate, actual.TestDate) {
		correct++
	} else {
		missed = append(missed, Miss{"test_date", dateStr(expected.TestDate), dateStr(actual.TestDate)})
	}
	total++

	if expected.OverallPass == actual.OverallPass {
		correct++
	} else {
		missed = append(missed, Miss{"overall_pass", boolStr(expected.OverallPass), boolStr(actual.OverallPass)})
	}
	total++

	// --- list fields: search for each expected entry ---
	for _, want := range expected.Cannabinoids {
		gotValue := 0.0
		nameFound := false
		for _, got := range actual.Cannabinoids {
			if norm(want.Name) == norm(got.Name) {
				gotValue = got.Value
				nameFound = true
				break
			}
		}
		if nameFound && floatEqual(want.Value, gotValue) {
			correct++
		} else {
			missed = append(missed, Miss{"cannabinoid: " + want.Name, numStr(want.Value), gotNum(gotValue, nameFound)})
		}
		total++
	}

	for _, want := range expected.Terpenes {
		gotValue := 0.0
		nameFound := false
		for _, got := range actual.Terpenes {
			if norm(want.Name) == norm(got.Name) {
				gotValue = got.Value
				nameFound = true
				break
			}
		}
		if nameFound && floatEqual(want.Value, gotValue) {
			correct++
		} else {
			missed = append(missed, Miss{"terpene: " + want.Name, numStr(want.Value), gotNum(gotValue, nameFound)})
		}
		total++
	}

	for _, want := range expected.Summary {
		gotStatus := ""
		nameFound := false
		for _, got := range actual.Summary {
			if norm(want.Name) == norm(got.Name) {
				gotStatus = got.Status
				nameFound = true
				break
			}
		}
		if nameFound && norm(want.Status) == norm(gotStatus) {
			correct++
		} else {
			missed = append(missed, Miss{"summary: " + want.Name, want.Status, gotStr(gotStatus, nameFound)})
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
	return dateStr(a) == dateStr(b)
}

func dateStr(t time.Time) string {
	return t.Format("2006-01-02")
}

func boolStr(b bool) string {
	return fmt.Sprintf("%v", b)
}

func numStr(f float64) string {
	return fmt.Sprintf("%.4g", f)
}

// gotNum / gotStr distinguish "the model gave a wrong value" from "the model
// never reported this entry at all".
func gotNum(f float64, nameFound bool) string {
	if !nameFound {
		return "(not found)"
	}
	return numStr(f)
}

func gotStr(s string, nameFound bool) string {
	if !nameFound {
		return "(not found)"
	}
	return s
}

// floatEqual treats two measured numbers as equal within a small tolerance, so
// rounding (25.34 vs 25.3) doesn't count as wrong. Loosen if the model is right
// but flagged.
func floatEqual(want, got float64) bool {
	return math.Abs(want-got) <= 0.02*math.Abs(want)+0.01
}
