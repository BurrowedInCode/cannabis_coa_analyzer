import type { Analysis, AnalysisSummary } from "@/types"
const BASE_URL = import.meta.env.VITE_API_URL

export async function analyzeCOA(files: File[]): Promise<void> {
  const formData = new FormData()

  if (files.length === 0) {
    throw new Error("no file selected")
  }
  files.forEach(file => { formData.append("coa", file) })

  const res = await fetch(`${BASE_URL}/coa/analyze`, {
    method: "POST",
    credentials: "include",
    body: formData,
  })

  if (!res.ok) {
    const message = await res.text()
    throw new Error(message)
  }

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

export async function getCOAAnalysis(id: string): Promise<Analysis> {
  const res = await fetch(`${BASE_URL}/coa/analyses/${id}`, {
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

export async function updateCOAAnalysis(id: string, analysis: Analysis): Promise<void> {
  const res = await fetch(`${BASE_URL}/coa/analyses/${id}`, {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
    },
    credentials: "include",
    body: JSON.stringify(analysis),
  })

  if (!res.ok) {
    const message = await res.text()
    throw new Error(message)
  }
}
