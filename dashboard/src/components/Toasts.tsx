import { X } from 'lucide-react'
import { useAppStore } from '../store'

export default function Toasts() {
  const toasts = useAppStore((s) => s.toasts)
  const removeToast = useAppStore((s) => s.removeToast)

  if (!toasts.length) return null

  return (
    <div style={{
      position: 'fixed', bottom: 16, right: 16, zIndex: 100,
      display: 'flex', flexDirection: 'column', gap: 6,
    }}>
      {toasts.map((t) => (
        <div key={t.id} style={{
          display: 'flex', alignItems: 'center', gap: 8,
          padding: '8px 12px',
          borderRadius: 8,
          border: '0.5px solid var(--border-primary)',
          background: 'var(--bg-primary)',
          fontSize: 12,
          color: t.type === 'error' ? '#dc2626' : t.type === 'success' ? '#1D9E75' : 'var(--text-primary)',
          boxShadow: '0 2px 8px rgba(0,0,0,0.08)',
          maxWidth: 320,
        }}>
          <span style={{ flex: 1 }}>{t.message}</span>
          <button
            onClick={() => removeToast(t.id)}
            style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--text-tertiary)', display: 'flex' }}
          >
            <X size={12} />
          </button>
        </div>
      ))}
    </div>
  )
}
