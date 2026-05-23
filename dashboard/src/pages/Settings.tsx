import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { ArrowLeft, Shield, RefreshCw, Key } from 'lucide-react'
import { changePassword, revokeAllSessions, getAuditLog } from '../lib/api'
import { useAppStore } from '../store'

export default function Settings() {
  const navigate = useNavigate()
  const user = useAppStore((s) => s.user)
  const serverInfo = useAppStore((s) => s.serverInfo)
  const addToast = useAppStore((s) => s.addToast)

  const [currentPw, setCurrentPw] = useState('')
  const [newPw, setNewPw] = useState('')
  const [confirmPw, setConfirmPw] = useState('')
  const [pwError, setPwError] = useState('')
  const [pwLoading, setPwLoading] = useState(false)

  const [auditLog, setAuditLog] = useState<{ timestamp: string; eventType: string; actor: string; detail: string; ipAddress: string }[]>([])
  const [auditLoading, setAuditLoading] = useState(false)

  async function handleChangePassword(e: React.FormEvent) {
    e.preventDefault()
    setPwError('')
    if (newPw !== confirmPw) { setPwError('Passwords do not match'); return }
    if (newPw.length < 12) { setPwError('Password must be at least 12 characters'); return }
    setPwLoading(true)
    try {
      await changePassword(currentPw, newPw)
      addToast('Password changed successfully', 'success')
      setCurrentPw(''); setNewPw(''); setConfirmPw('')
    } catch (err: unknown) {
      setPwError(err instanceof Error ? err.message : 'Failed to change password')
    } finally {
      setPwLoading(false)
    }
  }

  async function handleRevokeSessions() {
    if (!window.confirm('Revoke all sessions? All users will be signed out.')) return
    try {
      await revokeAllSessions()
      addToast('All sessions revoked', 'success')
    } catch (err: unknown) {
      addToast(err instanceof Error ? err.message : 'Failed to revoke sessions', 'error')
    }
  }

  async function handleLoadAuditLog() {
    setAuditLoading(true)
    try {
      const data = await getAuditLog()
      setAuditLog(data)
    } catch (err: unknown) {
      addToast(err instanceof Error ? err.message : 'Failed to load audit log', 'error')
    } finally {
      setAuditLoading(false)
    }
  }

  return (
    <div className="min-h-screen bg-[#0f1117] text-white">
      <div className="max-w-2xl mx-auto py-10 px-6">
        {/* Back */}
        <button
          onClick={() => navigate('/')}
          className="flex items-center gap-2 text-gray-400 hover:text-white mb-8 text-sm transition-colors"
        >
          <ArrowLeft size={16} /> Back to dashboard
        </button>

        <h1 className="text-2xl font-bold mb-8">Settings</h1>

        {/* Relay info */}
        <section className="rounded-xl border border-gray-800 bg-gray-900 p-6 mb-6">
          <h2 className="font-semibold text-gray-200 mb-4 flex items-center gap-2">
            <Shield size={16} /> Relay info
          </h2>
          <dl className="space-y-2 text-sm">
            <div className="flex justify-between">
              <dt className="text-gray-500">Version</dt>
              <dd className="text-gray-200">{serverInfo?.version ?? '—'}</dd>
            </div>
            <div className="flex justify-between">
              <dt className="text-gray-500">Uptime</dt>
              <dd className="text-gray-200">{serverInfo ? formatUptime(serverInfo.uptime) : '—'}</dd>
            </div>
            <div className="flex justify-between">
              <dt className="text-gray-500">Connected clients</dt>
              <dd className="text-gray-200">{serverInfo?.connectedClients ?? '—'}</dd>
            </div>
            <div className="flex justify-between">
              <dt className="text-gray-500">Logged in as</dt>
              <dd className="text-gray-200">{user?.username} ({user?.role})</dd>
            </div>
          </dl>
        </section>

        {/* Change password */}
        <section className="rounded-xl border border-gray-800 bg-gray-900 p-6 mb-6">
          <h2 className="font-semibold text-gray-200 mb-4 flex items-center gap-2">
            <Key size={16} /> Change password
          </h2>
          <form onSubmit={handleChangePassword} className="space-y-3">
            <div>
              <label className="block text-sm text-gray-400 mb-1">Current password</label>
              <input
                type="password"
                value={currentPw}
                onChange={(e) => setCurrentPw(e.target.value)}
                required
                className="w-full rounded-lg bg-gray-800 border border-gray-700 px-3 py-2 text-white focus:outline-none focus:ring-2 focus:ring-brand"
              />
            </div>
            <div>
              <label className="block text-sm text-gray-400 mb-1">New password</label>
              <input
                type="password"
                value={newPw}
                onChange={(e) => setNewPw(e.target.value)}
                required
                className="w-full rounded-lg bg-gray-800 border border-gray-700 px-3 py-2 text-white focus:outline-none focus:ring-2 focus:ring-brand"
              />
            </div>
            <div>
              <label className="block text-sm text-gray-400 mb-1">Confirm new password</label>
              <input
                type="password"
                value={confirmPw}
                onChange={(e) => setConfirmPw(e.target.value)}
                required
                className="w-full rounded-lg bg-gray-800 border border-gray-700 px-3 py-2 text-white focus:outline-none focus:ring-2 focus:ring-brand"
              />
            </div>
            {pwError && <p className="text-sm text-red-400">{pwError}</p>}
            <button
              type="submit"
              disabled={pwLoading}
              className="rounded-lg bg-brand px-4 py-2 text-sm font-medium text-white hover:bg-blue-500 transition-colors disabled:opacity-50"
            >
              {pwLoading ? 'Saving…' : 'Change password'}
            </button>
          </form>
        </section>

        {/* Danger zone */}
        <section className="rounded-xl border border-red-900/50 bg-gray-900 p-6 mb-6">
          <h2 className="font-semibold text-red-400 mb-4 flex items-center gap-2">
            <RefreshCw size={16} /> Danger zone
          </h2>
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-200 font-medium">Revoke all sessions</p>
              <p className="text-xs text-gray-500 mt-0.5">All users will be signed out immediately</p>
            </div>
            <button
              onClick={handleRevokeSessions}
              className="rounded-lg border border-red-700 px-4 py-2 text-sm text-red-400 hover:bg-red-900/30 transition-colors"
            >
              Revoke all
            </button>
          </div>
        </section>

        {/* Audit log */}
        <section className="rounded-xl border border-gray-800 bg-gray-900 p-6">
          <div className="flex items-center justify-between mb-4">
            <h2 className="font-semibold text-gray-200">Audit log</h2>
            <button
              onClick={handleLoadAuditLog}
              disabled={auditLoading}
              className="text-xs text-brand hover:underline disabled:opacity-50"
            >
              {auditLoading ? 'Loading…' : 'Load latest 100'}
            </button>
          </div>
          {auditLog.length === 0 ? (
            <p className="text-xs text-gray-600 italic">Click "Load latest 100" to view events</p>
          ) : (
            <div className="space-y-1 max-h-80 overflow-y-auto">
              {auditLog.map((entry, i) => (
                <div key={i} className="flex gap-3 text-xs py-1 border-b border-gray-800">
                  <span className="text-gray-500 flex-shrink-0 w-36">
                    {new Date(entry.timestamp).toLocaleString()}
                  </span>
                  <span className="text-brand font-mono flex-shrink-0 w-36">{entry.eventType}</span>
                  <span className="text-gray-300 truncate">{entry.detail}</span>
                </div>
              ))}
            </div>
          )}
        </section>
      </div>
    </div>
  )
}

function formatUptime(seconds: number): string {
  const d = Math.floor(seconds / 86400)
  const h = Math.floor((seconds % 86400) / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  const parts = []
  if (d) parts.push(`${d}d`)
  if (h) parts.push(`${h}h`)
  parts.push(`${m}m`)
  return parts.join(' ')
}
