// API client with automatic token refresh

let accessToken: string | null = null

export function setAccessToken(token: string | null) {
  accessToken = token
}

export function getAccessToken() {
  return accessToken
}

const REFRESH_TOKEN_KEY = 'taildog_refresh_token'

export function saveRefreshToken(token: string) {
  localStorage.setItem(REFRESH_TOKEN_KEY, token)
}

export function loadRefreshToken(): string | null {
  return localStorage.getItem(REFRESH_TOKEN_KEY)
}

export function clearRefreshToken() {
  localStorage.removeItem(REFRESH_TOKEN_KEY)
}

type FetchOptions = RequestInit & { skipAuth?: boolean }

export async function apiFetch(path: string, options: FetchOptions = {}): Promise<Response> {
  const { skipAuth, ...fetchOptions } = options
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(fetchOptions.headers as Record<string, string>),
  }

  if (!skipAuth && accessToken) {
    headers['Authorization'] = `Bearer ${accessToken}`
  }

  const res = await fetch(`/api${path}`, { ...fetchOptions, headers, credentials: 'include' })

  // Try silent refresh on 401
  if (res.status === 401 && !skipAuth) {
    const refreshed = await silentRefresh()
    if (refreshed) {
      headers['Authorization'] = `Bearer ${accessToken}`
      return fetch(`/api${path}`, { ...fetchOptions, headers, credentials: 'include' })
    }
  }

  return res
}

export async function silentRefresh(): Promise<boolean> {
  const refreshToken = loadRefreshToken()
  if (!refreshToken) return false
  try {
    const res = await fetch('/api/auth/refresh', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refreshToken }),
    })
    if (!res.ok) { clearRefreshToken(); return false }
    const data = await res.json()
    setAccessToken(data.accessToken)
    if (data.refreshToken) saveRefreshToken(data.refreshToken)
    scheduleRefresh(data.expiresIn)
    return true
  } catch {
    return false
  }
}

let refreshTimer: ReturnType<typeof setTimeout> | null = null

export function scheduleRefresh(expiresInSeconds: number) {
  if (refreshTimer) clearTimeout(refreshTimer)
  // Refresh 60 seconds before expiry
  const delay = Math.max((expiresInSeconds - 60) * 1000, 0)
  refreshTimer = setTimeout(async () => {
    await silentRefresh()
  }, delay)
}

export function clearRefreshTimer() {
  if (refreshTimer) {
    clearTimeout(refreshTimer)
    refreshTimer = null
  }
}

// ── Auth endpoints ──────────────────────────────────────────────────────────

export async function login(username: string, password: string) {
  const res = await apiFetch('/auth/login', {
    method: 'POST',
    body: JSON.stringify({ username, password }),
    skipAuth: true,
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'Login failed' }))
    throw new Error(err.error || 'Login failed')
  }
  return res.json() as Promise<{ accessToken: string; refreshToken: string; expiresIn: number }>
}

export async function logout() {
  const refreshToken = loadRefreshToken()
  await apiFetch('/auth/logout', {
    method: 'POST',
    body: JSON.stringify({ refreshToken: refreshToken ?? '' }),
  })
  clearRefreshTimer()
  setAccessToken(null)
  clearRefreshToken()
}

export async function getMe() {
  const res = await apiFetch('/auth/me')
  if (!res.ok) throw new Error('Unauthorized')
  return res.json() as Promise<{ id: string; username: string; role: string; lastLogin: string }>
}

// ── Data endpoints ──────────────────────────────────────────────────────────

export async function getClients() {
  const res = await apiFetch('/clients')
  if (!res.ok) throw new Error('Failed to fetch clients')
  return res.json()
}

export async function getServerInfo() {
  const res = await apiFetch('/server/info')
  if (!res.ok) throw new Error('Failed to fetch server info')
  return res.json()
}

export async function createTunnel(payload: {
  clientID: string
  protocol: string
  localPort: number
  remotePort: number
}) {
  const res = await apiFetch('/tunnels', { method: 'POST', body: JSON.stringify(payload) })
  if (!res.ok) throw new Error('Failed to create tunnel')
  return res.json()
}

export async function deleteTunnel(id: string) {
  const res = await apiFetch(`/tunnels/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error('Failed to delete tunnel')
}

export async function changePassword(currentPassword: string, newPassword: string) {
  const res = await apiFetch('/admin/change-password', {
    method: 'POST',
    body: JSON.stringify({ currentPassword, newPassword }),
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'Password change failed' }))
    throw new Error(err.error || 'Password change failed')
  }
}

export async function revokeAllSessions() {
  const res = await apiFetch('/admin/revoke-all-sessions', { method: 'POST' })
  if (!res.ok) throw new Error('Failed to revoke sessions')
}

export async function getAuditLog() {
  const res = await apiFetch('/admin/audit-log')
  if (!res.ok) throw new Error('Failed to fetch audit log')
  return res.json()
}

// ── Canvas persistence ──────────────────────────────────────────────────────

export async function getCanvasState(): Promise<Record<string, unknown>> {
  const res = await apiFetch('/canvas')
  if (!res.ok) return {}
  return res.json()
}

export async function saveCanvasState(state: Record<string, unknown>): Promise<void> {
  await apiFetch('/canvas', {
    method: 'PUT',
    body: JSON.stringify(state),
  })
}
