export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

type AuthHooks = {
  getToken: () => string | null
  onUnauthorized: () => void
}

let hooks: AuthHooks = { getToken: () => null, onUnauthorized: () => {} }

export function configureAuth(h: AuthHooks) {
  hooks = h
}

/**
 * Typed fetch wrapper: attaches the Bearer token, maps non-2xx to an
 * ApiError with the backend's `error` message, and on 401 clears the token
 * and lets the auth provider bounce to /login.
 */
export async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const headers: Record<string, string> = {}
  const isForm = body instanceof FormData
  if (body !== undefined && !isForm) headers['Content-Type'] = 'application/json'
  const token = hooks.getToken()
  if (token) headers['Authorization'] = `Bearer ${token}`

  const res = await fetch(path, {
    method,
    headers,
    body: body !== undefined ? (isForm ? (body as FormData) : JSON.stringify(body)) : undefined,
  })

  if (res.status === 401) {
    hooks.onUnauthorized()
    throw new ApiError(401, 'unauthorized')
  }

  if (!res.ok) {
    let message = `HTTP ${res.status}`
    try {
      const data: unknown = await res.json()
      if (data && typeof data === 'object' && typeof (data as { error?: unknown }).error === 'string') {
        message = (data as { error: string }).error
      }
    } catch {
      /* fall back to the status text */
    }
    throw new ApiError(res.status, message)
  }

  if (res.status === 204) return undefined as T
  return (await res.json()) as T
}
