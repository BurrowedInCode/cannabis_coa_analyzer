package coa

import "time"

type Cannabinoid struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
}

type Terpene struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
}

type TestSummary struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type Analysis struct {
	LabName      string        `json:"lab_name"`
	License      string        `json:"license"`
	SampleName   string        `json:"sample_name"`
	BatchNumber  string        `json:"batch_number"`
	SampleMatrix string        `json:"sample_matrix"`
	TestDate     time.Time     `json:"test_date"`
	OverallPass  bool          `json:"overall_pass"`
	Cannabinoids []Cannabinoid `json:"cannabinoids"`
	Terpenes     []Terpene     `json:"terpenes"`
	Summary      []TestSummary `json:"summary"`
}
