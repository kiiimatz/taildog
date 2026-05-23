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
    <div className="flex min-h-screen items-center justify-center bg-[#0f1117]">
      <div className="w-full max-w-sm rounded-2xl border border-gray-800 bg-gray-900 p-8 shadow-2xl">
        {/* Logo */}
        <div className="mb-8 text-center">
          <div className="text-4xl mb-2">🐕</div>
          <h1 className="text-2xl font-bold text-white tracking-tight">taildog</h1>
        </div>

        {isLocked ? (
          <div className="rounded-lg bg-red-900/30 border border-red-700 p-4 text-center">
            <p className="text-red-400 font-medium">Too many attempts</p>
            <p className="text-red-300 text-sm mt-1">
              Try again in {Math.floor(secondsLeft / 60)}m {secondsLeft % 60}s
            </p>
          </div>
        ) : (
          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-400 mb-1">Username</label>
              <input
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                autoComplete="username"
                required
                disabled={loading}
                className="w-full rounded-lg bg-gray-800 border border-gray-700 px-3 py-2 text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-brand focus:border-transparent disabled:opacity-50"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-400 mb-1">Password</label>
              <div className="relative">
                <input
                  type={showPassword ? 'text' : 'password'}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  autoComplete="current-password"
                  required
                  disabled={loading}
                  className="w-full rounded-lg bg-gray-800 border border-gray-700 px-3 py-2 pr-10 text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-brand focus:border-transparent disabled:opacity-50"
                />
                <button
                  type="button"
                  onClick={() => setShowPassword((v) => !v)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-white"
                  tabIndex={-1}
                >
                  {showPassword ? <EyeOff size={16} /> : <Eye size={16} />}
                </button>
              </div>
            </div>

            {error && (
              <p className="text-sm text-red-400 rounded-lg bg-red-900/20 border border-red-800 px-3 py-2">
                {error}
              </p>
            )}

            {attempts > 0 && attempts < MAX_ATTEMPTS && !error && (
              <p className="text-xs text-yellow-500">
                {MAX_ATTEMPTS - attempts} attempt{MAX_ATTEMPTS - attempts !== 1 ? 's' : ''} remaining
              </p>
            )}

            <button
              type="submit"
              disabled={loading || !username || !password}
              className="w-full rounded-lg bg-brand py-2.5 text-white font-medium hover:bg-blue-500 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {loading ? 'Signing in…' : 'Sign in'}
            </button>
          </form>
        )}

        <div className="mt-6 text-center">
          <span className="text-xs text-gray-600">── taildog v0.1.0 ──</span>
        </div>
      </div>
    </div>
  )
}
