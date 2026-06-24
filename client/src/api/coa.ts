import type { Analysis, AnalysisSummary } from "@/types"
const BASE_URL = import.meta.env.VITE_API_URL

export async function analyzeCOA(file: File | null): Promise<Analysis> {
  if (!file) {
    throw new Error("no file selected")
  }

  const formData = new FormData()
  formData.append("coa", file)

  const res = await fetch(`${BASE_URL}/coa/analyze`, {
    method: "POST",
    credentials: "include",
    body: formData,
  })

  if (!res.ok) {
    const message = await res.text()
    throw new Error(message)
  }

  return res.json()
}

export async function getAllCOAAnalyses(limit: number, offset: number): Promise<AnalysisSummary[]> {
  const res = await fetch(`${BASE_URL}/coa/analyses?limit=${limit}&offset=${offset}`, {
    method: "GET",
    headers: {
      Accept: "application/json"
    },
    credentials: "include",
  })

  if (!res.ok) {
    const message = await res.text()
    throw new Error(message)
  }

  return res.json()
}

