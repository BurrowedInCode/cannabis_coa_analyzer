package eval

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/BurrowedInCode/cannabis_coa_analyzer/internal/coa"
)

type Miss struct {
	Field    string
	Expected string
	Got      string
}

// Function to compare LLM output against JSON golden data
func Score(expected, actual *coa.Analysis) (correct int, total int, missed []Miss) {
	if norm(expected.Laboratory.Name) == norm(actual.Laboratory.Name) {
		correct++
	} else {
		missed = append(missed, Miss{Field: "laboratory.name", Expected: expected.Laboratory.Name, Got: actual.Laboratory.Name})
	}
	total++

	if norm(expected.Laboratory.Address) == norm(actual.Laboratory.Address) {
		correct++
	} else {
		missed = append(missed, Miss{Field: "laboratory.address", Expected: expected.Laboratory.Address, Got: actual.Laboratory.Address})
	}
	total++

	if norm(expected.Laboratory.Phone) == norm(actual.Laboratory.Phone) {
		correct++
	} else {
		missed = append(missed, Miss{Field: "laboratory.phone", Expected: expected.Laboratory.Phone, Got: actual.Laboratory.Phone})
	}
	total++

	if norm(expected.Laboratory.Certification) == norm(actual.Laboratory.Certification) {
		correct++
	} else {
		missed = append(missed, Miss{Field: "laboratory.certification", Expected: expected.Laboratory.Certification, Got: actual.Laboratory.Certification})
	}
	total++

	if norm(expected.SampleName) == norm(actual.SampleName) {
		correct++
	} else {
		missed = append(missed, Miss{Field: "sample_name", Expected: expected.SampleName, Got: actual.SampleName})
	}
	total++

	if norm(expected.SeedToSaleNumber) == norm(actual.SeedToSaleNumber) {
		correct++
	} else {
		missed = append(missed, Miss{Field: "seed_to_sale_number", Expected: expected.SeedToSaleNumber, Got: actual.SeedToSaleNumber})
	}
	total++

	if norm(expected.SampleMatrix) == norm(actual.SampleMatrix) {
		correct++
	} else {
		missed = append(missed, Miss{Field: "sample_matrix", Expected: expected.SampleMatrix, Got: actual.SampleMatrix})
	}
	total++

	expectedTestDate := dateToString(expected.TestDate)
	actualTestDate := dateToString(actual.TestDate)

	if expectedTestDate == actualTestDate {
		correct++
	} else {
		missed = append(missed, Miss{Field: "test_date", Expected: expectedTestDate, Got: actualTestDate})
	}
	total++

	if expected.OverallPass == actual.OverallPass {
		correct++
	} else {
		missed = append(missed, Miss{Field: "overall_pass", Expected: strconv.FormatBool(expected.OverallPass), Got: strconv.FormatBool(actual.OverallPass)})
	}
	total++

	for _, want := range expected.Cannabinoids {
		var gotValue float64
		var found bool
		for _, got := range actual.Cannabinoids {
			if norm(want.Name) == norm(got.Name) {
				gotValue = got.Value
				found = true
				break
			}
		}
		if found && floatEqual(want.Value, gotValue) {
			correct++
		} else {
			missed = append(missed, Miss{Field: "Cannabinoid: " + want.Name, Expected: floatStr(want.Value), Got: gotNum(gotValue, found)})
		}
		total++
	}

	for _, want := range expected.Terpenes {
		var gotValue float64
		var found bool
		for _, got := range actual.Terpenes {
			if norm(want.Name) == norm(got.Name) {
				gotValue = got.Value
				found = true
				break
			}
		}
		if found && floatEqual(want.Value, gotValue) {
			correct++
		} else {
			missed = append(missed, Miss{Field: "Terpene: " + want.Name, Expected: floatStr(want.Value), Got: gotNum(gotValue, found)})
		}
		total++
	}

	for _, want := range expected.Summary {
		var gotValue string
		var found bool
		for _, got := range actual.Summary {
			if norm(want.Name) == norm(got.Name) {
				gotValue = got.Status
				found = true
				break
			}
		}
		if found && norm(want.Status) == norm(gotValue) {
			correct++
		} else {
			missed = append(missed, Miss{Field: "Summary: " + want.Name, Expected: want.Status, Got: gotStr(gotValue, found)})
		}
		total++
	}

	return correct, total, missed
}

func norm(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}

// Makes comparing dates easier
func dateToString(d time.Time) string {
	return d.Format("2006-01-02")
}

func floatEqual(want, got float64) bool {
	return math.Abs(want-got) <= 0.02*math.Abs(want)+0.01
}

func floatStr(f float64) string {
	return fmt.Sprintf("%.4g", f)
}

func gotNum(got float64, found bool) string {
	if !found {
		return "(not found)"
	}
	return floatStr(got)
}

func gotStr(got string, found bool) string {
	if !found {
		return "(not found)"
	}
	return got
}
