import { CheckCircle2 } from 'lucide-react'
import { useAppStore } from '../store'

export default function Sidebar() {
  const clients = useAppStore((s) => s.clients)
  const canvasClients = useAppStore((s) => s.canvasClients)
  const onCanvasIDs = new Set(canvasClients.map((c) => c.clientID))

  function onDragStart(e: React.DragEvent, clientID: string) {
    e.dataTransfer.setData('clientID', clientID)
    e.dataTransfer.effectAllowed = 'copy'
  }

  return (
    <aside style={{
      width: 240,
      flexShrink: 0,
      display: 'flex',
      flexDirection: 'column',
      background: 'var(--bg-secondary)',
      borderRight: '0.5px solid var(--border-primary)',
    }}>
      {/* Header */}
      <div style={{
        padding: '14px 14px 10px',
        borderBottom: '0.5px solid var(--border-primary)',
      }}>
        <span style={{
          fontSize: 11,
          fontWeight: 500,
          color: 'var(--text-tertiary)',
          letterSpacing: '0.05em',
          textTransform: 'uppercase',
        }}>
          Connected clients
        </span>
      </div>

      {/* Client list */}
      <div style={{ flex: 1, overflowY: 'auto', padding: 8 }}>
        {clients.length === 0 ? (
          <p style={{ fontSize: 11, color: 'var(--text-tertiary)', padding: '8px 10px', fontStyle: 'italic' }}>
            No clients connected
          </p>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
            {clients.map((client) => {
              const onCanvas = onCanvasIDs.has(client.id)
              return (
                <ClientRow
                  key={client.id}
                  name={client.name}
                  ip={client.ip}
                  online={client.online}
                  dimmed={onCanvas}
                  draggable={!onCanvas}
                  onDragStart={(e) => !onCanvas && onDragStart(e, client.id)}
                />
              )
            })}
          </div>
        )}
      </div>

      {/* Footer status */}
      <div style={{
        padding: '12px 14px',
        borderTop: '0.5px solid var(--border-primary)',
      }}>
        <span style={{
          display: 'inline-flex',
          alignItems: 'center',
          gap: 5,
          fontSize: 11,
          color: 'var(--text-success)',
          background: 'var(--bg-success)',
          padding: '3px 8px',
          borderRadius: 20,
        }}>
          <CheckCircle2 size={12} />
          taildog up
        </span>
      </div>
    </aside>
  )
}

function ClientRow({
  name, ip, online, dimmed, draggable, onDragStart,
}: {
  name: string
  ip: string
  online: boolean
  dimmed: boolean
  draggable: boolean
  onDragStart: (e: React.DragEvent) => void
}) {
  return (
    <div
      draggable={draggable}
      onDragStart={onDragStart}
      title={dimmed ? 'Already on canvas' : 'Drag to canvas'}
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 8,
        padding: '8px 10px',
        borderRadius: 8,
        border: '0.5px solid transparent',
        cursor: draggable ? 'grab' : 'default',
        opacity: dimmed ? 0.4 : 1,
        transition: 'background 0.1s, border-color 0.1s',
      }}
      onMouseEnter={(e) => {
        if (!dimmed) {
          const el = e.currentTarget as HTMLDivElement
          el.style.background = 'var(--bg-primary)'
          el.style.borderColor = 'var(--border-secondary)'
        }
      }}
      onMouseLeave={(e) => {
        const el = e.currentTarget as HTMLDivElement
        el.style.background = 'transparent'
        el.style.borderColor = 'transparent'
      }}
    >
      {/* Status dot */}
      <div style={{
        width: 7,
        height: 7,
        borderRadius: '50%',
        flexShrink: 0,
        background: online ? '#1D9E75' : '#888780',
      }} />
      {/* Text */}
      <div style={{ minWidth: 0 }}>
        <div style={{ fontSize: 12, fontWeight: 500, color: 'var(--text-primary)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
          {name}
        </div>
        <div style={{ fontSize: 10, color: 'var(--text-tertiary)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
          {ip}
        </div>
      </div>
    </div>
  )
}
