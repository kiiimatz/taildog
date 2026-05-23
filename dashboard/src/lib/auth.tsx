import React, { createContext, useContext, useEffect, useState } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { getMe, setAccessToken, silentRefresh, scheduleRefresh, clearRefreshTimer } from './api'
import { useAppStore } from '../store'

interface AuthContextValue {
  loading: boolean
  initialized: boolean
}

const AuthContext = createContext<AuthContextValue>({ loading: true, initialized: false })

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [loading, setLoading] = useState(true)
  const [initialized, setInitialized] = useState(false)
  const setUser = useAppStore((s) => s.setUser)
  const navigate = useNavigate()
  const location = useLocation()

  useEffect(() => {
    async function bootstrap() {
      // Attempt silent refresh from httpOnly cookie
      const ok = await silentRefresh()
      if (ok) {
        try {
          const me = await getMe()
          setUser(me)
          scheduleRefresh(900)
          setInitialized(true)
        } catch {
          setAccessToken(null)
          clearRefreshTimer()
          setUser(null)
        }
      } else {
        setUser(null)
      }
      setLoading(false)
    }
    bootstrap()
  }, [])

  useEffect(() => {
    // Redirect unauthenticated users
    if (!loading) {
      const user = useAppStore.getState().user
      if (!user && location.pathname !== '/login') {
        navigate('/login', { replace: true })
      }
    }
  }, [loading])

  return (
    <AuthContext.Provider value={{ loading, initialized }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  return useContext(AuthContext)
}

export function PrivateRoute({ children, adminOnly }: { children: React.ReactNode; adminOnly?: boolean }) {
  const user = useAppStore((s) => s.user)
  const { loading } = useAuth()
  const navigate = useNavigate()

  useEffect(() => {
    if (!loading && !user) navigate('/login', { replace: true })
    if (!loading && adminOnly && user?.role !== 'admin') navigate('/', { replace: true })
  }, [loading, user])

  if (loading) return <div className="flex h-screen items-center justify-center text-gray-400">Loading…</div>
  if (!user) return null
  if (adminOnly && user.role !== 'admin') return null
  return <>{children}</>
}
