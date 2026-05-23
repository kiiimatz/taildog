import { MoreHorizontal } from 'lucide-react'
import { useAppStore } from '../store'

export default function Sidebar() {
  const clients = useAppStore((s) => s.clients)
  const canvasClients = useAppStore((s) => s.canvasClients)

  function onDragStart(e: React.DragEvent, clientID: string) {
    e.dataTransfer.setData('clientID', clientID)
    e.dataTransfer.effectAllowed = 'copy'
  }

  const onCanvasIDs = new Set(canvasClients.map((c) => c.clientID))

  return (
    <aside
      className="flex-shrink-0 flex flex-col"
      style={{
        width: 230,
        background: '#111119',
        borderRight: '1px solid rgba(255,255,255,0.06)',
      }}
    >
      {/* Header */}
      <div
        className="flex items-center justify-between px-4 pt-5 pb-3"
        style={{ borderBottom: '1px solid rgba(255,255,255,0.05)' }}
      >
        <span
          className="text-[10px] font-semibold uppercase tracking-widest"
          style={{ color: '#3d3f52', letterSpacing: '0.12em' }}
        >
          Connected clients
        </span>
        <button
          className="flex items-center justify-center rounded-md w-6 h-6 transition-colors"
          style={{ color: '#3d3f52' }}
          onMouseEnter={(e) => { (e.currentTarget as HTMLButtonElement).style.color = '#6b7280' }}
          onMouseLeave={(e) => { (e.currentTarget as HTMLButtonElement).style.color = '#3d3f52' }}
          title="More options"
        >
          <MoreHorizontal size={14} />
        </button>
      </div>

      {/* Client list */}
      <div className="flex-1 overflow-y-auto px-2 py-2">
        {clients.length === 0 ? (
          <p
            className="text-xs italic px-3 py-2"
            style={{ color: '#3d3f52' }}
          >
            No clients connected
          </p>
        ) : (
          <div className="space-y-0.5">
            {clients.map((client) => {
              const isOnCanvas = onCanvasIDs.has(client.id)
              return (
                <div
                  key={client.id}
                  draggable={!isOnCanvas}
                  onDragStart={(e) => !isOnCanvas && onDragStart(e, client.id)}
                  className="flex items-center gap-3 px-3 py-2.5 rounded-lg transition-colors"
                  style={{
                    cursor: isOnCanvas ? 'default' : 'grab',
                    opacity: isOnCanvas ? 0.4 : 1,
                  }}
                  onMouseEnter={(e) => {
                    if (!isOnCanvas) {
                      (e.currentTarget as HTMLDivElement).style.background = '#1c1d2a'
                    }
                  }}
                  onMouseLeave={(e) => {
                    (e.currentTarget as HTMLDivElement).style.background = 'transparent'
                  }}
                  title={isOnCanvas ? 'Already on canvas' : 'Drag to canvas'}
                >
                  {/* Status dot */}
                  <div
                    className="w-2 h-2 rounded-full flex-shrink-0"
                    style={{
                      background: client.online ? '#22c55e' : '#3d3f52',
                      boxShadow: client.online ? '0 0 4px #22c55e66' : 'none',
                    }}
                  />
                  {/* Info */}
                  <div className="min-w-0">
                    <p
                      className="text-sm font-medium truncate"
                      style={{ color: '#e8e9f0' }}
                    >
                      {client.name}
                    </p>
                    <p className="text-xs truncate" style={{ color: '#6b7280' }}>
                      {client.ip}
                    </p>
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </div>

      {/* Bottom status bar */}
      <div
        className="px-4 py-3 flex items-center gap-2"
        style={{ borderTop: '1px solid rgba(255,255,255,0.06)' }}
      >
        <span style={{ color: '#22c55e', fontSize: 14 }}>⊙</span>
        <span className="text-xs font-medium" style={{ color: '#22c55e' }}>
          taildog up
        </span>
      </div>
    </aside>
  )
}
