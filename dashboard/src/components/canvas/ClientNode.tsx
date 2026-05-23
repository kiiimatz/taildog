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
      className="relative"
      style={{
        minWidth: 200,
        borderRadius: 12,
        border: '1px solid rgba(255,255,255,0.08)',
        background: '#161622',
        boxShadow: '0 4px 20px rgba(0,0,0,0.4)',
      }}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
    >
      {/* Right handle — source */}
      <Handle
        type="source"
        position={Position.Right}
        style={{
          width: 14,
          height: 14,
          borderRadius: '50%',
          border: '2px solid #378ADD',
          background: '#ffffff',
          right: -7,
        }}
      />

      {/* Remove button */}
      {hovered && (
        <button
          onClick={() => d.onRemove(d.clientID)}
          className="absolute -top-2 -right-2 z-10 rounded-full p-0.5 transition-colors"
          style={{
            background: '#242535',
            border: '1px solid rgba(255,255,255,0.1)',
            color: '#6b7280',
          }}
          onMouseEnter={(e) => {
            (e.currentTarget as HTMLButtonElement).style.background = '#7f1d1d'
            ;(e.currentTarget as HTMLButtonElement).style.color = '#fca5a5'
          }}
          onMouseLeave={(e) => {
            (e.currentTarget as HTMLButtonElement).style.background = '#242535'
            ;(e.currentTarget as HTMLButtonElement).style.color = '#6b7280'
          }}
        >
          <X size={11} />
        </button>
      )}

      <div className="px-4 py-3">
        {/* Label */}
        <p
          className="text-[10px] font-semibold uppercase mb-1"
          style={{ color: '#3d3f52', letterSpacing: '0.1em' }}
        >
          Client
        </p>

        {/* Hostname row */}
        <div className="flex items-center gap-2 mb-0.5">
          <div
            className="w-2 h-2 rounded-full flex-shrink-0"
            style={{
              background: d.online ? '#22c55e' : '#3d3f52',
              boxShadow: d.online ? '0 0 4px #22c55e66' : 'none',
            }}
          />
          <p
            className="font-semibold text-sm truncate"
            style={{ color: '#e8e9f0' }}
          >
            {d.name}
          </p>
        </div>

        {/* IP */}
        <p className="text-xs mb-3 pl-4" style={{ color: '#6b7280' }}>
          {d.ip}
        </p>

        {/* Port badges */}
        {d.ports.length > 0 && (
          <div className="flex flex-wrap gap-1">
            {d.ports.map((p) => (
              <span
                key={`${p.protocol}-${p.port}`}
                className="rounded-md px-2 py-0.5 text-[10px] font-mono"
                style={{
                  background: '#1e1f2e',
                  color: '#9ca3af',
                  border: '1px solid rgba(255,255,255,0.07)',
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

export default memo(ClientNode)
