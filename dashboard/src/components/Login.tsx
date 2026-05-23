import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { Eye, EyeOff } from 'lucide-react'
import { login, scheduleRefresh, setAccessToken, saveRefreshToken, getMe } from '../lib/api'
import { useAppStore } from '../store'

const LOCKOUT_MS = 10 * 60 * 1000
const MAX_ATTEMPTS = 5

const inputStyle: React.CSSProperties = {
  width: '100%',
  fontSize: 13,
  padding: '7px 10px',
  border: '0.5px solid var(--border-secondary)',
  borderRadius: 8,
  background: 'var(--bg-secondary)',
  color: 'var(--text-primary)',
  outline: 'none',
}

export default function Login() {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [showPw, setShowPw] = useState(false)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [attempts, setAttempts] = useState(0)
  const [lockedUntil, setLockedUntil] = useState<number | null>(null)
  const [secondsLeft, setSecondsLeft] = useState(0)
  const setUser = useAppStore((s) => s.setUser)
  const navigate = useNavigate()

  const locked = lockedUntil !== null && Date.now() < lockedUntil

  useEffect(() => {
    if (!locked) return
    const iv = setInterval(() => {
      const left = Math.ceil(((lockedUntil ?? 0) - Date.now()) / 1000)
      if (left <= 0) { setLockedUntil(null); setAttempts(0) }
      else setSecondsLeft(left)
    }, 500)
    return () => clearInterval(iv)
  }, [locked, lockedUntil])

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (locked || loading) return
    setError('')
    setLoading(true)
    try {
      const data = await login(username, password)
      setAccessToken(data.accessToken)
      saveRefreshToken(data.refreshToken)
      scheduleRefresh(data.expiresIn)
      const me = await getMe()
      setUser(me)
      navigate('/', { replace: true })
    } catch (err: unknown) {
      const n = attempts + 1
      setAttempts(n)
      if (n >= MAX_ATTEMPTS) {
        setLockedUntil(Date.now() + LOCKOUT_MS)
        setSecondsLeft(LOCKOUT_MS / 1000)
      } else {
        setError(err instanceof Error ? err.message : 'Invalid credentials')
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={{
      minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center',
      background: 'var(--bg-tertiary)',
    }}>
      <div style={{
        width: 300,
        background: 'var(--bg-primary)',
        border: '0.5px solid var(--border-primary)',
        borderRadius: 12,
        padding: '28px 24px',
      }}>
        {/* Logo */}
        <div style={{ textAlign: 'center', marginBottom: 22 }}>
          <div style={{ fontSize: 22, marginBottom: 6 }}>🐕</div>
          <div style={{ fontSize: 16, fontWeight: 600, color: 'var(--text-primary)' }}>taildog</div>
          <div style={{ fontSize: 11, color: 'var(--text-tertiary)', marginTop: 2 }}>Sign in to continue</div>
        </div>

        {locked ? (
          <div style={{
            padding: '10px 12px', borderRadius: 8, textAlign: 'center',
            background: 'rgba(220,38,38,0.06)', border: '0.5px solid rgba(220,38,38,0.2)',
          }}>
            <div style={{ fontSize: 12, fontWeight: 500, color: '#dc2626' }}>Too many attempts</div>
            <div style={{ fontSize: 11, color: '#ef4444', marginTop: 3 }}>
              Try again in {Math.floor(secondsLeft / 60)}m {secondsLeft % 60}s
            </div>
          </div>
        ) : (
          <form onSubmit={handleSubmit}>
            {/* Username */}
            <div style={{ marginBottom: 10 }}>
              <label style={{ display: 'block', fontSize: 11, color: 'var(--text-secondary)', marginBottom: 4 }}>
                Username
              </label>
              <input
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                autoComplete="username"
                required
                disabled={loading}
                style={{ ...inputStyle, opacity: loading ? 0.6 : 1 }}
                onFocus={(e) => { (e.currentTarget as HTMLInputElement).style.borderColor = '#378ADD' }}
                onBlur={(e) => { (e.currentTarget as HTMLInputElement).style.borderColor = 'var(--border-secondary)' }}
              />
            </div>

            {/* Password */}
            <div style={{ marginBottom: 12 }}>
              <label style={{ display: 'block', fontSize: 11, color: 'var(--text-secondary)', marginBottom: 4 }}>
                Password
              </label>
              <div style={{ position: 'relative' }}>
                <input
                  type={showPw ? 'text' : 'password'}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  autoComplete="current-password"
                  required
                  disabled={loading}
                  style={{ ...inputStyle, paddingRight: 32, opacity: loading ? 0.6 : 1 }}
                  onFocus={(e) => { (e.currentTarget as HTMLInputElement).style.borderColor = '#378ADD' }}
                  onBlur={(e) => { (e.currentTarget as HTMLInputElement).style.borderColor = 'var(--border-secondary)' }}
                />
                <button
                  type="button"
                  onClick={() => setShowPw((v) => !v)}
                  tabIndex={-1}
                  style={{
                    position: 'absolute', right: 8, top: '50%', transform: 'translateY(-50%)',
                    background: 'none', border: 'none', cursor: 'pointer',
                    color: 'var(--text-tertiary)', display: 'flex', alignItems: 'center',
                  }}
                >
                  {showPw ? <EyeOff size={14} /> : <Eye size={14} />}
                </button>
              </div>
            </div>

            {error && (
              <div style={{
                fontSize: 11, padding: '6px 10px', borderRadius: 6, marginBottom: 10,
                color: '#dc2626', background: 'rgba(220,38,38,0.06)', border: '0.5px solid rgba(220,38,38,0.2)',
              }}>
                {error}
              </div>
            )}

            {attempts > 0 && attempts < MAX_ATTEMPTS && !error && (
              <div style={{ fontSize: 11, color: '#d97706', marginBottom: 10 }}>
                {MAX_ATTEMPTS - attempts} attempt{MAX_ATTEMPTS - attempts !== 1 ? 's' : ''} remaining
              </div>
            )}

            <button
              type="submit"
              disabled={loading || !username || !password}
              style={{
                width: '100%', padding: '7px 0', borderRadius: 8,
                border: '0.5px solid #185FA5', background: '#185FA5',
                color: '#fff', fontSize: 13, fontWeight: 500,
                cursor: loading || !username || !password ? 'not-allowed' : 'pointer',
                opacity: loading || !username || !password ? 0.5 : 1,
              }}
              onMouseEnter={(e) => {
                if (!loading && username && password)
                  (e.currentTarget as HTMLButtonElement).style.background = '#1a6bbf'
              }}
              onMouseLeave={(e) => {
                (e.currentTarget as HTMLButtonElement).style.background = '#185FA5'
              }}
            >
              {loading ? 'Signing in…' : 'Sign in'}
            </button>
          </form>
        )}

        <div style={{ textAlign: 'center', marginTop: 16, fontSize: 10, color: 'var(--text-tertiary)' }}>
          taildog v0.1.0
        </div>
      </div>
    </div>
  )
}
