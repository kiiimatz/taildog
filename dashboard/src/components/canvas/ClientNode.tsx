import { memo, useState } from 'react'
import { Handle, Position, type NodeProps } from '@xyflow/react'
import { Monitor, X } from 'lucide-react'

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
      className="relative min-w-[200px] rounded-xl border border-gray-700 bg-gray-900 shadow-lg"
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
    >
      <Handle
        type="source"
        position={Position.Right}
        className="!w-4 !h-4 !rounded-full !border-2 !border-brand !bg-gray-900"
      />

      {hovered && (
        <button
          onClick={() => d.onRemove(d.clientID)}
          className="absolute -top-2 -right-2 z-10 rounded-full bg-gray-700 hover:bg-red-600 p-0.5 text-white transition-colors"
        >
          <X size={12} />
        </button>
      )}

      <div className="px-4 py-3">
        <p className="text-[10px] uppercase tracking-widest text-gray-500 mb-1">client</p>
        <div className="flex items-center gap-2 mb-1">
          <div className={`w-2 h-2 rounded-full flex-shrink-0 ${d.online ? 'bg-green-400' : 'bg-gray-500'}`} />
          <Monitor size={14} className="text-gray-400" />
          <span className="font-semibold text-white text-sm truncate">{d.name}</span>
        </div>
        <p className="text-xs text-gray-400 mb-3">{d.ip}</p>

        {d.ports.length > 0 && (
          <div className="flex flex-wrap gap-1">
            {d.ports.map((p) => (
              <span
                key={`${p.protocol}-${p.port}`}
                className="rounded-full bg-gray-800 border border-gray-700 px-2 py-0.5 text-[10px] text-gray-300 font-mono"
              >
                {p.protocol}:{p.port}
              </span>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

export default memo(ClientNode)
