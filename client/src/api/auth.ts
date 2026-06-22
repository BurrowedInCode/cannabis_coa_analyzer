const BASE_URL = import.meta.env.VITE_API_URL

export async function UserLogin(username: string, password: string): Promise<void> {
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
