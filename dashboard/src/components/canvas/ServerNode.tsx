import { memo, useState } from 'react'
import { Handle, Position, type NodeProps } from '@xyflow/react'

function formatIP(ip: string) {
  if (!ip) return '—'
  const clean = ip.replace(/^\[|\]$/g, '')
  if (clean === '::1' || clean === '127.0.0.1' || clean === 'localhost') return 'localhost'
  return clean
}

export interface ServerNodeData extends Record<string, unknown> {
  name: string
  ip: string
  ports: { port: number; protocol: string }[]
}

function ServerNode({ data }: NodeProps) {
  const d = data as unknown as ServerNodeData
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
        type="target"
        position={Position.Left}
        style={{
          width: 10, height: 10,
          borderRadius: '50%',
          background: 'var(--bg-primary)',
          border: '1.5px solid #378ADD',
          cursor: 'crosshair',
          left: -6,
        }}
      />

      {/* Accent dot (server indicator) */}
      <div style={{
        width: 7, height: 7,
        borderRadius: '50%',
        flexShrink: 0,
        background: '#378ADD',
      }} />

      {/* Text */}
      <div style={{ minWidth: 0 }}>
        <div style={{ fontSize: 12, fontWeight: 500, color: 'var(--text-primary)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
          {d.name || 'renode-relay'}
        </div>
        <div style={{ fontSize: 10, color: 'var(--text-tertiary)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
          {formatIP(d.ip)}
        </div>
      </div>
    </div>
  )
}

export default memo(ServerNode)
