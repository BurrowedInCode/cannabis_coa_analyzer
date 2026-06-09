package coa

type Cannabinoid struct {
	Name  string
	Value float64
	Unit  string
}

type Terpene struct {
	Name  string
	Value float64
	Unit  string
}

type TestSummary struct {
	Name   string
	Status string
}

type Analysis struct {
	LabName      string
	License      string
	SampleName   string
	BatchNumber  string
	SampleMatrix string
	TestDate     string
	OverallPass  bool
	Cannabinoids []Cannabinoid
	Terpenes     []Terpene
	Summary      []TestSummary
}
