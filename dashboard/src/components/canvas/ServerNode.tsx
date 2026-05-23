import { memo } from 'react'
import { Handle, Position, type NodeProps } from '@xyflow/react'
import { Server } from 'lucide-react'

export interface ServerNodeData extends Record<string, unknown> {
  name: string
  ip: string
  ports: { port: number; protocol: string }[]
}

function ServerNode({ data }: NodeProps) {
  const d = data as unknown as ServerNodeData
  return (
    <div className="relative min-w-[200px] rounded-xl border border-brand/40 bg-gray-900 shadow-lg shadow-brand/10">
      <Handle
        type="target"
        position={Position.Left}
        className="!w-4 !h-4 !rounded-full !border-2 !border-brand !bg-gray-900"
      />

      <div className="px-4 py-3">
        <p className="text-[10px] uppercase tracking-widest text-gray-500 mb-1">server</p>
        <div className="flex items-center gap-2 mb-2">
          <Server size={14} className="text-brand" />
          <span className="font-semibold text-white text-sm">{d.name || 'relay'}</span>
        </div>
        <p className="text-xs text-gray-400 mb-3">{d.ip || '—'}</p>

        {d.ports.length > 0 && (
          <div className="flex flex-wrap gap-1">
            {d.ports.map((p) => (
              <span
                key={`${p.protocol}-${p.port}`}
                className="rounded-full bg-brand/10 border border-brand/30 px-2 py-0.5 text-[10px] text-brand font-mono"
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

export default memo(ServerNode)
