import { memo, useState } from 'react'
import { Handle, Position, type NodeProps } from '@xyflow/react'
import { X } from 'lucide-react'

export interface ClientNodeData extends Record<string, unknown> {
  clientID: string
  name: string
  ip: string
  online: boolean
  ports: { port: number; protocol: string }[]
  onRemove: (clientID: string) => void
}

function ClientNode({ data }: NodeProps) {
  const d = data as unknown as ClientNodeData
  const [hovered, setHovered] = useState(false)

  return (
    <div
      style={{
        minWidth: 150,
        background: 'var(--bg-primary)',
        border: '0.5px solid var(--border-secondary)',
        borderRadius: 12,
        padding: '12px 14px',
        cursor: 'move',
        userSelect: 'none',
        position: 'relative',
      }}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
    >
      {/* Right handle */}
      <Handle
        type="source"
        position={Position.Right}
        style={{
          width: 18, height: 18,
          borderRadius: '50%',
          background: 'var(--bg-primary)',
          border: '1.5px solid #378ADD',
          cursor: 'crosshair',
          right: -10,
        }}
      />

      {/* Remove button */}
      {hovered && (
        <button
          onClick={() => d.onRemove(d.clientID)}
          style={{
            position: 'absolute',
            top: -8, right: -8,
            width: 18, height: 18,
            borderRadius: '50%',
            background: 'var(--bg-primary)',
            border: '0.5px solid var(--border-primary)',
            color: 'var(--text-secondary)',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            cursor: 'pointer',
            zIndex: 10,
          }}
          onMouseEnter={(e) => {
            (e.currentTarget as HTMLButtonElement).style.background = '#fee2e2'
            ;(e.currentTarget as HTMLButtonElement).style.color = '#dc2626'
          }}
          onMouseLeave={(e) => {
            (e.currentTarget as HTMLButtonElement).style.background = 'var(--bg-primary)'
            ;(e.currentTarget as HTMLButtonElement).style.color = 'var(--text-secondary)'
          }}
        >
          <X size={10} />
        </button>
      )}

      {/* Label */}
      <div style={{ fontSize: 10, fontWeight: 500, color: 'var(--text-tertiary)', textTransform: 'uppercase', letterSpacing: '0.05em', marginBottom: 4 }}>
        client
      </div>

      {/* Name + status dot */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 5 }}>
        <div style={{
          width: 7, height: 7, borderRadius: '50%', flexShrink: 0,
          background: d.online ? '#1D9E75' : '#888780',
        }} />
        <div style={{ fontSize: 13, fontWeight: 500, color: 'var(--text-primary)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
          {d.name}
        </div>
      </div>

      {/* IP */}
      <div style={{ fontSize: 11, color: 'var(--text-secondary)', marginTop: 1, paddingLeft: 12 }}>
        {d.ip}
      </div>

      {/* Port badges */}
      {d.ports.length > 0 && (
        <div style={{ marginTop: 6, paddingTop: 6, borderTop: '0.5px solid var(--border-secondary)', display: 'flex', flexWrap: 'wrap', gap: 3 }}>
          {d.ports.map((p) => (
            <span key={`${p.protocol}-${p.port}`} style={{
              fontSize: 10,
              padding: '1px 6px',
              borderRadius: 4,
              background: 'var(--bg-info)',
              color: 'var(--text-info)',
              fontFamily: 'monospace',
            }}>
              :{p.port}
            </span>
          ))}
        </div>
      )}
    </div>
  )
}

export default memo(ClientNode)
