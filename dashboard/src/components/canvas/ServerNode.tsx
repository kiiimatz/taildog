import { memo } from 'react'
import { Handle, Position, type NodeProps } from '@xyflow/react'

export interface ServerNodeData extends Record<string, unknown> {
  name: string
  ip: string
  ports: { port: number; protocol: string }[]
}

function ServerNode({ data }: NodeProps) {
  const d = data as unknown as ServerNodeData

  return (
    <div style={{
      minWidth: 150,
      background: 'var(--bg-primary)',
      border: '1.5px solid #378ADD',
      borderRadius: 12,
      padding: '12px 14px',
      cursor: 'move',
      userSelect: 'none',
      position: 'relative',
    }}>
      {/* Left handle */}
      <Handle
        type="target"
        position={Position.Left}
        style={{
          width: 18, height: 18,
          borderRadius: '50%',
          background: '#E6F1FB',
          border: '1.5px solid #378ADD',
          cursor: 'crosshair',
          left: -10,
        }}
      />

      {/* Label */}
      <div style={{ fontSize: 10, fontWeight: 500, color: 'var(--text-tertiary)', textTransform: 'uppercase', letterSpacing: '0.05em', marginBottom: 4 }}>
        server
      </div>

      {/* Name */}
      <div style={{ fontSize: 13, fontWeight: 500, color: 'var(--text-primary)' }}>
        {d.name || 'taildog-relay'}
      </div>

      {/* IP */}
      <div style={{ fontSize: 11, color: 'var(--text-secondary)', marginTop: 1 }}>
        {d.ip || '—'}
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

export default memo(ServerNode)
