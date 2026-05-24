const API_URL = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080'

export function getApiUrl(): string {
  return API_URL
}

export async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${API_URL}${path}`, {
    ...init,
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      ...init?.headers,
    },
  })

  if (!res.ok) {
    const err = new Error(`API error: ${res.status}`) as Error & { status: number }
    err.status = res.status
    throw err
  }

  if (!res.headers.get('Content-Type')?.includes('application/json')) return undefined as T
  return res.json() as Promise<T>
}

export async function checkHealth(): Promise<boolean> {
  try {
    await apiFetch<void>('/health')
    return true
  } catch {
    return false
  }
}
