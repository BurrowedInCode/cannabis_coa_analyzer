export interface Laboratory {
  name: string
  address: string
  phone: string
  certification: string
}

export interface Cannabinoid {
  name: string
  value: number
  unit: string
}

export interface Terpene {
  name: string
  value: number
  unit: string
}

export interface TestSummary {
  name: string
  status: string
}

export interface Analysis {
  laboratory: Laboratory
  sample_name: string
  seed_to_sale_number: string
  sample_matrix: string
  test_date: string
  overall_pass: boolean
  cannabinoids: Cannabinoid[]
  terpenes: Terpene[]
  summary: TestSummary[]
}

export interface AnalysisSummary {
  id: string,
  sample_name: string,
  seed_to_sale_number: string,
  test_date: string,
  overall_pass: boolean
}



