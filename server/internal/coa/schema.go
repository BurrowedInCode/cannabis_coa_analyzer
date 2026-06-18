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

type Laboratory struct {
	Name          string `json:"name"`
	Address       string `json:"address"`
	Phone         string `json:"phone"`
	Certification string `json:"certification"`
}

type Analysis struct {
	Laboratory       Laboratory    `json:"laboratory"`
	SampleName       string        `json:"sample_name"`
	SeedToSaleNumber string        `json:"seed_to_sale_number"`
	SampleMatrix     string        `json:"sample_matrix"`
	TestDate         time.Time     `json:"test_date"`
	OverallPass      bool          `json:"overall_pass"`
	Cannabinoids     []Cannabinoid `json:"cannabinoids"`
	Terpenes         []Terpene     `json:"terpenes"`
	Summary          []TestSummary `json:"summary"`
}
