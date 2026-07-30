import type { Me } from "@/types"

const BASE_URL = import.meta.env.VITE_API_URL

export async function login(username: string, password: string): Promise<void> {
  const res = await fetch(`${BASE_URL}/user/login`, {
    method: "POST",
    credentials: "include",
    body: JSON.stringify({ username: username, password: password })
  })
  if (!res.ok) {
    const message = await res.text()
    throw new Error(message)
  }
}

export async function getMe(): Promise<Me> {
  const res = await fetch(`${BASE_URL}/user/me`, {
    method: "GET",
    headers: { Accept: "application/json" },
    credentials: "include",
  })

  if (!res.ok) {
    throw new Error("unauthenticated")
  }
  return res.json()
}

export async function logout(): Promise<void> {
  const res = await fetch(`${BASE_URL}/user/logout`, {
    method: "POST",
    credentials: "include",
  })

  if (!res.ok) {
    const message = await res.text()
    throw new Error(message)
  }
}

export async function register(username: string, password: string): Promise<void> {
  const res = await fetch(`${BASE_URL}/user/register`, {
    method: "POST",
    body: JSON.stringify({ username: username, password: password })
  })
  if (!res.ok) {
    const message = await res.text()
    throw new Error(message)
  }
}
