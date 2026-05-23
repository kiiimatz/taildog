import { memo, useState } from 'react'
import { Handle, Position, type NodeProps } from '@xyflow/react'
import { X } from 'lucide-react'

function formatIP(ip: string) {
  if (!ip) return '—'
  const clean = ip.replace(/^\[|\]$/g, '') // strip brackets if any
  if (clean === '::1' || clean === '127.0.0.1' || clean === 'localhost') return 'localhost'
  return clean
}

export interface ClientNodeData extends Record<string, unknown> {
  instanceID: string
  clientID: string
  name: string
  ip: string
  online: boolean
  ports: { port: number; protocol: string }[]
  onRemove: (instanceID: string) => void
}

function ClientNode({ data }: NodeProps) {
  const d = data as unknown as ClientNodeData
  const [hovered, setHovered] = useState(false)

  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 8,
        padding: '8px 10px',
        borderRadius: 8,
        background: 'var(--bg-primary)',
        border: `0.5px solid ${hovered ? 'var(--border-primary)' : 'var(--border-secondary)'}`,
        cursor: 'move',
        userSelect: 'none',
        position: 'relative',
        minWidth: 140,
      }}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
    >
      <Handle
        type="source"
        position={Position.Right}
        style={{
          width: 10, height: 10,
          borderRadius: '50%',
          background: 'var(--bg-primary)',
          border: '1.5px solid #378ADD',
          cursor: 'crosshair',
          right: -6,
        }}
      />

      {hovered && (
        <button
          onClick={() => d.onRemove(d.instanceID)}
          style={{
            position: 'absolute',
            top: -7, right: -7,
            width: 16, height: 16,
            borderRadius: '50%',
            background: 'var(--bg-primary)',
            border: '0.5px solid var(--border-primary)',
            color: 'var(--text-tertiary)',
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
            ;(e.currentTarget as HTMLButtonElement).style.color = 'var(--text-tertiary)'
          }}
        >
          <X size={9} />
        </button>
      )}

      {/* Status dot */}
      <div style={{
        width: 7, height: 7,
        borderRadius: '50%',
        flexShrink: 0,
        background: d.online ? '#1D9E75' : '#888780',
      }} />

      {/* Text */}
      <div style={{ minWidth: 0 }}>
        <div style={{ fontSize: 12, fontWeight: 500, color: 'var(--text-primary)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
          {d.name}
        </div>
        <div style={{ fontSize: 10, color: 'var(--text-tertiary)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
          {formatIP(d.ip)}
        </div>
      </div>
    </div>
  )
}

export default memo(ClientNode)
