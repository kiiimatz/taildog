import { Settings, LogOut } from 'lucide-react'
import { Link } from 'react-router-dom'
import { useNavigate } from 'react-router-dom'
import { logout } from '../lib/api'
import { useAppStore } from '../store'

export default function Topbar() {
  const user = useAppStore((s) => s.user)
  const setUser = useAppStore((s) => s.setUser)
  const serverInfo = useAppStore((s) => s.serverInfo)
  const navigate = useNavigate()

  async function handleLogout() {
    await logout()
    setUser(null)
    navigate('/login', { replace: true })
  }

  const isOnline = true // relay ws is connected

  return (
    <header className="flex items-center justify-between bg-gray-950 border-b border-gray-800 px-4 py-2 h-12">
      <div className="flex items-center gap-3">
        <span className="text-xs text-gray-500">
          {serverInfo?.serverName ?? 'relay'} · {serverInfo?.serverIP ?? ''}
        </span>
        <span
          className={`flex items-center gap-1 text-xs px-2 py-0.5 rounded-full ${
            isOnline ? 'bg-green-900/40 text-green-400' : 'bg-gray-800 text-gray-500'
          }`}
        >
          <span className={`w-1.5 h-1.5 rounded-full ${isOnline ? 'bg-green-400' : 'bg-gray-500'}`} />
          {isOnline ? 'up' : 'down'}
        </span>
      </div>

      <div className="flex items-center gap-2">
        <span className="text-xs text-gray-500">{user?.username}</span>
        {user?.role === 'admin' && (
          <Link
            to="/settings"
            className="rounded-lg p-1.5 text-gray-400 hover:text-white hover:bg-gray-800 transition-colors"
            title="Settings"
          >
            <Settings size={16} />
          </Link>
        )}
        <button
          onClick={handleLogout}
          className="rounded-lg p-1.5 text-gray-400 hover:text-white hover:bg-gray-800 transition-colors"
          title="Log out"
        >
          <LogOut size={16} />
        </button>
      </div>
    </header>
  )
}
