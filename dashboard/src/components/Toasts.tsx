import { X } from 'lucide-react'
import { useAppStore } from '../store'

export default function Toasts() {
  const toasts = useAppStore((s) => s.toasts)
  const removeToast = useAppStore((s) => s.removeToast)

  if (!toasts.length) return null

  return (
    <div className="fixed bottom-4 right-4 z-50 flex flex-col gap-2">
      {toasts.map((t) => (
        <div
          key={t.id}
          className={`flex items-center gap-3 rounded-lg border px-4 py-3 text-sm shadow-lg max-w-sm ${
            t.type === 'error'
              ? 'bg-red-950 border-red-700 text-red-200'
              : t.type === 'success'
              ? 'bg-green-950 border-green-700 text-green-200'
              : 'bg-gray-900 border-gray-700 text-gray-200'
          }`}
        >
          <span className="flex-1">{t.message}</span>
          <button onClick={() => removeToast(t.id)} className="text-current opacity-60 hover:opacity-100">
            <X size={14} />
          </button>
        </div>
      ))}
    </div>
  )
}
