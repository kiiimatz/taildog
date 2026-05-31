import { useState } from 'react'
import { ChevronDown, ChevronRight, CheckCircle2, Trash2 } from 'lucide-react'
import { useAppStore } from '../store'

const sectionLabel: React.CSSProperties = {
  fontSize: 10,
  fontWeight: 700,
  letterSpacing: '0.1em',
  textTransform: 'uppercase',
  color: 'var(--text-tertiary)',
  fontFamily: "ui-monospace, 'Cascadia Code', 'SF Mono', 'Fira Code', monospace",
}

export default function Sidebar() {
  const clients = useAppStore((s) => s.clients)
  const isDraggingCanvasNode = useAppStore((s) => s.isDraggingCanvasNode)
  const [clientsOpen, setClientsOpen] = useState(true)
  const [blocksOpen, setBlocksOpen] = useState(true)

  function onDragStart(e: React.DragEvent, clientID: string) {
    e.dataTransfer.setData('clientID', clientID)
    e.dataTransfer.effectAllowed = 'copy'
  }

  function onBlockDragStart(e: React.DragEvent, blockType: string) {
    e.dataTransfer.setData('blockType', blockType)
    e.dataTransfer.effectAllowed = 'copy'
  }

  return (
    <aside data-sidebar="true" style={{
      position: 'absolute',
      top: 16,
      left: 16,
      width: 240,
      height: 'calc(100vh - 32px)',
      display: 'flex',
      flexDirection: 'column',
      background: 'var(--sidebar-bg)',
      backdropFilter: 'blur(20px) saturate(1.8)',
      WebkitBackdropFilter: 'blur(20px) saturate(1.8)',
      borderRadius: 14,
      border: '0.5px solid var(--sidebar-border)',
      boxShadow: 'var(--sidebar-shadow)',
      overflow: 'hidden',
      zIndex: 10,
    }}>

      {/* ── CONNECTED CLIENTS ─────────────────────────────────────────── */}
      <div style={{ borderBottom: '0.5px solid var(--border-primary)' }}>
        <button
          onClick={() => setClientsOpen((v) => !v)}
          style={{
            width: '100%', display: 'flex', alignItems: 'center', justifyContent: 'space-between',
            padding: '12px 14px',
            background: 'none', border: 'none', cursor: 'pointer',
          }}
          onMouseEnter={(e) => { (e.currentTarget as HTMLButtonElement).style.background = 'var(--bg-tertiary)' }}
          onMouseLeave={(e) => { (e.currentTarget as HTMLButtonElement).style.background = 'none' }}
        >
          <span style={sectionLabel}>Connected Clients</span>
          {clientsOpen
            ? <ChevronDown size={11} color="var(--text-tertiary)" />
            : <ChevronRight size={11} color="var(--text-tertiary)" />}
        </button>

        {clientsOpen && (
          <div style={{ padding: '0 8px 8px', display: 'flex', flexDirection: 'column', gap: 3 }}>
            {clients.length === 0 ? (
              <p style={{ fontSize: 11, color: 'var(--text-tertiary)', padding: '6px 10px', fontStyle: 'italic' }}>
                No clients connected
              </p>
            ) : (
              clients.map((client) => (
                <ClientRow
                  key={client.id}
                  name={client.name}
                  ip={client.ip}
                  online={client.online}
                  onDragStart={(e) => onDragStart(e, client.id)}
                />
              ))
            )}
          </div>
        )}
      </div>

      {/* ── BLOCKS ────────────────────────────────────────────────────── */}
      <div style={{ borderBottom: '0.5px solid var(--border-primary)' }}>
        <button
          onClick={() => setBlocksOpen((v) => !v)}
          style={{
            width: '100%', display: 'flex', alignItems: 'center', justifyContent: 'space-between',
            padding: '12px 14px',
            background: 'none', border: 'none', cursor: 'pointer',
          }}
          onMouseEnter={(e) => { (e.currentTarget as HTMLButtonElement).style.background = 'var(--bg-tertiary)' }}
          onMouseLeave={(e) => { (e.currentTarget as HTMLButtonElement).style.background = 'none' }}
        >
          <span style={sectionLabel}>Blocks</span>
          {blocksOpen
            ? <ChevronDown size={11} color="var(--text-tertiary)" />
            : <ChevronRight size={11} color="var(--text-tertiary)" />}
        </button>

        {blocksOpen && (
          <div style={{ padding: '0 8px 8px', display: 'flex', flexDirection: 'column', gap: 3 }}>
            <BlockItem
              label="Tunnel"
              description="port forwarding"
              onDragStart={(e) => onBlockDragStart(e, 'tunnel')}
            />
          </div>
        )}
      </div>

      {/* ── Spacer ────────────────────────────────────────────────────── */}
      <div style={{ flex: 1 }} />

      {/* ── Footer ────────────────────────────────────────────────────── */}
      <div style={{ padding: '12px 14px', borderTop: '0.5px solid var(--border-primary)' }}>
        <span style={{
          display: 'inline-flex', alignItems: 'center', gap: 5,
          fontSize: 11, color: 'var(--text-success)', background: 'var(--bg-success)',
          padding: '3px 8px', borderRadius: 20,
        }}>
          <CheckCircle2 size={12} />
          renode up
        </span>
      </div>
      {/* ── Drop-to-remove overlay (shown while dragging a canvas node) ── */}
      {isDraggingCanvasNode && (
        <div style={{
          position: 'absolute', inset: 8, zIndex: 20, pointerEvents: 'none',
          borderRadius: 10,
          border: '1.5px dashed rgba(55,138,221,0.5)',
          background: 'rgba(55,138,221,0.06)',
          display: 'flex', flexDirection: 'column',
          alignItems: 'center', justifyContent: 'center', gap: 6,
        }}>
          <Trash2 size={16} color="rgba(55,138,221,0.7)" />
          <span style={{ fontSize: 11, color: 'rgba(55,138,221,0.8)', fontWeight: 500 }}>
            Drop to remove
          </span>
        </div>
      )}
    </aside>
  )
}

// ── Client row ──────────────────────────────────────────────────────────────

function ClientRow({ name, ip, online, onDragStart }: {
  name: string; ip: string; online: boolean
  onDragStart: (e: React.DragEvent) => void
}) {
  return (
    <div
      draggable
      onDragStart={onDragStart}
      title="Drag to canvas"
      style={{
        display: 'flex', alignItems: 'center', gap: 8,
        padding: '8px 10px', borderRadius: 8,
        border: '0.5px solid transparent', cursor: 'grab',
        transition: 'background 0.1s, border-color 0.1s',
      }}
      onMouseEnter={(e) => {
        const el = e.currentTarget as HTMLDivElement
        el.style.background = 'var(--bg-primary)'
        el.style.borderColor = 'var(--border-secondary)'
      }}
      onMouseLeave={(e) => {
        const el = e.currentTarget as HTMLDivElement
        el.style.background = 'transparent'
        el.style.borderColor = 'transparent'
      }}
    >
      <div style={{ width: 7, height: 7, borderRadius: '50%', flexShrink: 0, background: online ? '#1D9E75' : '#888780' }} />
      <div style={{ minWidth: 0 }}>
        <div style={{ fontSize: 12, fontWeight: 500, color: 'var(--text-primary)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{name}</div>
        <div style={{ fontSize: 10, color: 'var(--text-tertiary)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{ip}</div>
      </div>
    </div>
  )
}

// ── Block item ──────────────────────────────────────────────────────────────

function BlockItem({ label, description, onDragStart }: {
  label: string; description: string
  onDragStart: (e: React.DragEvent) => void
}) {
  return (
    <div
      draggable
      onDragStart={onDragStart}
      title={`Drag "${label}" to canvas`}
      style={{
        display: 'flex', alignItems: 'center', gap: 8,
        padding: '8px 10px', borderRadius: 8,
        border: '0.5px solid transparent', cursor: 'grab',
        transition: 'background 0.1s, border-color 0.1s',
      }}
      onMouseEnter={(e) => {
        const el = e.currentTarget as HTMLDivElement
        el.style.background = 'var(--bg-primary)'
        el.style.borderColor = 'var(--border-secondary)'
      }}
      onMouseLeave={(e) => {
        const el = e.currentTarget as HTMLDivElement
        el.style.background = 'transparent'
        el.style.borderColor = 'transparent'
      }}
    >
      {/* Block icon */}
      <div style={{
        width: 24, height: 24, borderRadius: 6, flexShrink: 0,
        background: 'var(--bg-info)', border: '0.5px solid rgba(55,138,221,0.25)',
        display: 'flex', alignItems: 'center', justifyContent: 'center',
      }}>
        <div style={{ width: 8, height: 8, borderRadius: 2, background: '#378ADD' }} />
      </div>
      <div style={{ minWidth: 0 }}>
        <div style={{ fontSize: 12, fontWeight: 500, color: 'var(--text-primary)' }}>{label}</div>
        <div style={{ fontSize: 10, color: 'var(--text-tertiary)' }}>{description}</div>
      </div>
    </div>
  )
}
