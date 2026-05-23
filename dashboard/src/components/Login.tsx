import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { Eye, EyeOff } from 'lucide-react'
import { login, scheduleRefresh } from '../lib/api'
import { getMe } from '../lib/api'
import { setAccessToken } from '../lib/api'
import { useAppStore } from '../store'

const LOCKOUT_DURATION_MS = 10 * 60 * 1000
const MAX_ATTEMPTS = 5

export default function Login() {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [attempts, setAttempts] = useState(0)
  const [lockedUntil, setLockedUntil] = useState<number | null>(null)
  const [secondsLeft, setSecondsLeft] = useState(0)
  const setUser = useAppStore((s) => s.setUser)
  const navigate = useNavigate()

  const isLocked = lockedUntil !== null && Date.now() < lockedUntil

  useEffect(() => {
    if (!isLocked) return
    const iv = setInterval(() => {
      const left = Math.ceil(((lockedUntil ?? 0) - Date.now()) / 1000)
      if (left <= 0) {
        setLockedUntil(null)
        setAttempts(0)
        clearInterval(iv)
      } else {
        setSecondsLeft(left)
      }
    }, 500)
    return () => clearInterval(iv)
  }, [isLocked, lockedUntil])

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (isLocked || loading) return
    setError('')
    setLoading(true)

    try {
      const data = await login(username, password)
      setAccessToken(data.accessToken)
      scheduleRefresh(data.expiresIn)
      const me = await getMe()
      setUser(me)
      navigate('/', { replace: true })
    } catch (err: unknown) {
      const newAttempts = attempts + 1
      setAttempts(newAttempts)
      if (newAttempts >= MAX_ATTEMPTS) {
        setLockedUntil(Date.now() + LOCKOUT_DURATION_MS)
        setSecondsLeft(LOCKOUT_DURATION_MS / 1000)
        setError('')
      } else {
        setError(err instanceof Error ? err.message : 'Invalid credentials')
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <div
      className="flex min-h-screen items-center justify-center"
      style={{ background: '#0c0d14' }}
    >
      <div
        className="w-full max-w-sm rounded-2xl p-8"
        style={{
          background: '#111119',
          border: '1px solid rgba(255,255,255,0.07)',
          boxShadow: '0 32px 80px rgba(0,0,0,0.6)',
        }}
      >
        {/* Logo / wordmark */}
        <div className="mb-8 text-center">
          <div
            className="inline-flex items-center justify-center w-12 h-12 rounded-2xl mb-4"
            style={{ background: 'rgba(55,138,221,0.12)', border: '1px solid rgba(55,138,221,0.2)' }}
          >
            <span style={{ color: '#378ADD', fontSize: 22 }}>⊙</span>
          </div>
          <h1
            className="text-xl font-semibold tracking-tight"
            style={{ color: '#e8e9f0' }}
          >
            taildog
          </h1>
          <p className="text-xs mt-1" style={{ color: '#3d3f52' }}>
            Sign in to continue
          </p>
        </div>

        {isLocked ? (
          <div
            className="rounded-xl p-4 text-center"
            style={{
              background: 'rgba(127,29,29,0.25)',
              border: '1px solid rgba(239,68,68,0.25)',
            }}
          >
            <p className="text-sm font-medium" style={{ color: '#fca5a5' }}>
              Too many attempts
            </p>
            <p className="text-sm mt-1" style={{ color: '#f87171' }}>
              Try again in {Math.floor(secondsLeft / 60)}m {secondsLeft % 60}s
            </p>
          </div>
        ) : (
          <form onSubmit={handleSubmit} className="space-y-4">
            {/* Username */}
            <div>
              <label
                className="block text-xs font-medium mb-1.5"
                style={{ color: '#6b7280' }}
              >
                Username
              </label>
              <input
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                autoComplete="username"
                required
                disabled={loading}
                className="w-full rounded-lg px-3 py-2.5 text-sm focus:outline-none disabled:opacity-50 transition-colors"
                style={{
                  background: '#0c0d14',
                  border: '1px solid rgba(255,255,255,0.08)',
                  color: '#e8e9f0',
                }}
                onFocus={(e) => {
                  (e.currentTarget as HTMLInputElement).style.borderColor = 'rgba(55,138,221,0.5)'
                }}
                onBlur={(e) => {
                  (e.currentTarget as HTMLInputElement).style.borderColor = 'rgba(255,255,255,0.08)'
                }}
              />
            </div>

            {/* Password */}
            <div>
              <label
                className="block text-xs font-medium mb-1.5"
                style={{ color: '#6b7280' }}
              >
                Password
              </label>
              <div className="relative">
                <input
                  type={showPassword ? 'text' : 'password'}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  autoComplete="current-password"
                  required
                  disabled={loading}
                  className="w-full rounded-lg px-3 py-2.5 pr-10 text-sm focus:outline-none disabled:opacity-50 transition-colors"
                  style={{
                    background: '#0c0d14',
                    border: '1px solid rgba(255,255,255,0.08)',
                    color: '#e8e9f0',
                  }}
                  onFocus={(e) => {
                    (e.currentTarget as HTMLInputElement).style.borderColor = 'rgba(55,138,221,0.5)'
                  }}
                  onBlur={(e) => {
                    (e.currentTarget as HTMLInputElement).style.borderColor = 'rgba(255,255,255,0.08)'
                  }}
                />
                <button
                  type="button"
                  onClick={() => setShowPassword((v) => !v)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 transition-colors"
                  style={{ color: '#3d3f52' }}
                  onMouseEnter={(e) => {
                    (e.currentTarget as HTMLButtonElement).style.color = '#6b7280'
                  }}
                  onMouseLeave={(e) => {
                    (e.currentTarget as HTMLButtonElement).style.color = '#3d3f52'
                  }}
                  tabIndex={-1}
                >
                  {showPassword ? <EyeOff size={15} /> : <Eye size={15} />}
                </button>
              </div>
            </div>

            {/* Error */}
            {error && (
              <p
                className="text-xs rounded-lg px-3 py-2"
                style={{
                  color: '#fca5a5',
                  background: 'rgba(127,29,29,0.2)',
                  border: '1px solid rgba(239,68,68,0.2)',
                }}
              >
                {error}
              </p>
            )}

            {/* Attempts warning */}
            {attempts > 0 && attempts < MAX_ATTEMPTS && !error && (
              <p className="text-xs" style={{ color: '#f59e0b' }}>
                {MAX_ATTEMPTS - attempts} attempt{MAX_ATTEMPTS - attempts !== 1 ? 's' : ''} remaining
              </p>
            )}

            {/* Submit */}
            <button
              type="submit"
              disabled={loading || !username || !password}
              className="w-full rounded-lg py-2.5 text-sm font-medium transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
              style={{ background: '#378ADD', color: '#ffffff' }}
              onMouseEnter={(e) => {
                if (!loading && username && password) {
                  (e.currentTarget as HTMLButtonElement).style.background = '#4a9de8'
                }
              }}
              onMouseLeave={(e) => {
                (e.currentTarget as HTMLButtonElement).style.background = '#378ADD'
              }}
            >
              {loading ? 'Signing in…' : 'Sign in'}
            </button>
          </form>
        )}

        <div className="mt-6 text-center">
          <span className="text-[11px]" style={{ color: '#3d3f52' }}>
            taildog v0.1.0
          </span>
        </div>
      </div>
    </div>
  )
}
