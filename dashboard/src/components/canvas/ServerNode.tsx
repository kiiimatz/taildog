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
    <div
      className="relative"
      style={{
        minWidth: 200,
        borderRadius: 12,
        border: '1.5px solid #378ADD',
        background: '#161622',
        boxShadow: '0 0 24px rgba(55,138,221,0.12)',
      }}
    >
      {/* Left handle — target */}
      <Handle
        type="target"
        position={Position.Left}
        style={{
          width: 14,
          height: 14,
          borderRadius: '50%',
          border: '2px solid #378ADD',
          background: '#ffffff',
          left: -7,
        }}
      />

      <div className="px-4 py-3">
        {/* Label */}
        <p
          className="text-[10px] font-semibold uppercase mb-1"
          style={{ color: '#3d3f52', letterSpacing: '0.1em' }}
        >
          Server
        </p>

        {/* Hostname */}
        <p
          className="font-semibold text-sm mb-0.5"
          style={{ color: '#e8e9f0' }}
        >
          {d.name || 'relay'}
        </p>

        {/* IP */}
        <p className="text-xs mb-3" style={{ color: '#6b7280' }}>
          {d.ip || '—'}
        </p>

        {/* Port badges */}
        {d.ports.length > 0 && (
          <div className="flex flex-wrap gap-1">
            {d.ports.map((p) => (
              <span
                key={`${p.protocol}-${p.port}`}
                className="rounded-md px-2 py-0.5 text-[10px] font-mono"
                style={{
                  background: '#0f1e36',
                  color: '#60a5fa',
                  border: '1px solid rgba(55,138,221,0.25)',
                }}
              >
                :{p.port}
              </span>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

export default memo(ServerNode)
