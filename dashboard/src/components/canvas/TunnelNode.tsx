import { memo } from 'react'
import { Handle, Position, type NodeProps } from '@xyflow/react'
import { X } from 'lucide-react'

export interface TunnelNodeData extends Record<string, unknown> {
  protocol: string
  localPort: string
  remotePort: string
  status: 'idle' | 'connecting' | 'active' | 'error'
  tunnelID?: string
  onDataChange: (id: string, field: string, value: string) => void
  onRemove: (id: string) => void
  onActivate: (id: string) => void
  onDeactivate: (id: string) => void
}

const PROTOCOLS = ['tcp', 'udp', 'http', 'https', 'socks5', 'quic', 'ws']

function TunnelNode({ data, id }: NodeProps) {
  const d = data as unknown as TunnelNodeData

  const isOn = d.status === 'active' || d.status === 'connecting'
  const isEditable = d.status === 'idle' || d.status === 'error'
  const switchDisabled = d.status === 'connecting' || (isEditable && !d.localPort)

  const statusColor =
    d.status === 'active'     ? '#1D9E75' :
    d.status === 'error'      ? '#dc2626' :
    d.status === 'connecting' ? '#378ADD' :
    'var(--border-primary)'

  const fieldStyle: React.CSSProperties = {
    fontSize: 10,
    padding: '2px 5px',
    border: '0.5px solid var(--border-secondary)',
    borderRadius: 4,
    background: 'var(--bg-secondary)',
    color: 'var(--text-primary)',
    outline: 'none',
    opacity: isEditable ? 1 : 0.45,
    minWidth: 0,
  }

  function handleSwitch() {
    if (switchDisabled) return
    if (isOn) d.onDeactivate(id)
    else d.onActivate(id)
  }

  return (
    <div style={{
      padding: '7px 9px',
      borderRadius: 8,
      background: 'var(--bg-primary)',
      border: `0.5px solid ${isOn ? statusColor : 'var(--border-secondary)'}`,
      minWidth: 200,
      position: 'relative',
      transition: 'border-color 0.3s',
    }}>
      {/* Handles */}
      <Handle
        type="target" position={Position.Left}
        style={{ width: 10, height: 10, borderRadius: '50%', background: 'var(--bg-primary)', border: '1.5px solid #378ADD', left: -6 }}
      />
      <Handle
        type="source" position={Position.Right}
        style={{ width: 10, height: 10, borderRadius: '50%', background: 'var(--bg-primary)', border: '1.5px solid #378ADD', right: -6 }}
      />

      {/* Remove button */}
      <button
        onClick={() => d.onRemove(id)}
        className="nodrag"
        style={{
          position: 'absolute', top: -6, right: -6,
          width: 14, height: 14, borderRadius: '50%',
          background: 'var(--bg-primary)', border: '0.5px solid var(--border-primary)',
          color: 'var(--text-tertiary)', display: 'flex', alignItems: 'center', justifyContent: 'center',
          cursor: 'pointer', zIndex: 10,
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
        <X size={8} />
      </button>

      {/* Header */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 5, marginBottom: 6 }}>
        <div style={{ width: 5, height: 5, borderRadius: '50%', background: statusColor, flexShrink: 0, transition: 'background 0.3s' }} />
        <span style={{
          fontSize: 9, fontWeight: 700, letterSpacing: '0.1em', textTransform: 'uppercase',
          color: 'var(--text-tertiary)', fontFamily: "ui-monospace, 'SF Mono', monospace",
          flex: 1,
        }}>
          tunnel
        </span>

        {/* Status label */}
        <span style={{ fontSize: 9, color: statusColor, transition: 'color 0.3s' }}>
          {d.status === 'active'     ? `:${d.remotePort}` :
           d.status === 'connecting' ? 'connecting…' :
           d.status === 'error'      ? 'port in use' : ''}
        </span>

        {/* Toggle switch */}
        <div
          onClick={handleSwitch}
          className="nodrag"
          title={isOn ? 'Turn off' : 'Turn on'}
          style={{
            width: 26, height: 14, borderRadius: 7, flexShrink: 0,
            background: isOn ? statusColor : 'var(--border-primary)',
            position: 'relative',
            cursor: switchDisabled ? 'not-allowed' : 'pointer',
            opacity: d.status === 'connecting' ? 0.6 : 1,
            transition: 'background 0.25s',
          }}
        >
          <div style={{
            position: 'absolute',
            top: 2,
            left: isOn ? 14 : 2,
            width: 10, height: 10, borderRadius: '50%',
            background: '#fff',
            transition: 'left 0.2s',
            boxShadow: '0 1px 2px rgba(0,0,0,0.25)',
          }} />
        </div>
      </div>

      {/* Inputs row */}
      <div style={{ display: 'flex', gap: 4, alignItems: 'center' }}>
        {/* Protocol */}
        <select
          value={d.protocol}
          onChange={(e) => d.onDataChange(id, 'protocol', e.target.value)}
          disabled={!isEditable}
          className="nodrag"
          style={{ ...fieldStyle, width: 54 }}
        >
          {PROTOCOLS.map((p) => <option key={p} value={p}>{p}</option>)}
        </select>

        {/* Client port */}
        <input
          type="number" value={d.localPort}
          onChange={(e) => d.onDataChange(id, 'localPort', e.target.value)}
          placeholder="client"
          min={1} max={65535}
          disabled={!isEditable}
          className="nodrag"
          style={{ ...fieldStyle, width: 54 }}
          onFocus={(e) => { (e.currentTarget as HTMLInputElement).style.borderColor = '#378ADD' }}
          onBlur={(e) => { (e.currentTarget as HTMLInputElement).style.borderColor = 'var(--border-secondary)' }}
        />

        {/* Server port */}
        <input
          type="number" value={d.remotePort}
          onChange={(e) => d.onDataChange(id, 'remotePort', e.target.value)}
          placeholder="server"
          min={1} max={65535}
          disabled={!isEditable}
          className="nodrag"
          style={{ ...fieldStyle, width: 54 }}
          onFocus={(e) => { (e.currentTarget as HTMLInputElement).style.borderColor = '#378ADD' }}
          onBlur={(e) => { (e.currentTarget as HTMLInputElement).style.borderColor = 'var(--border-secondary)' }}
        />
      </div>
    </div>
  )
}

export default memo(TunnelNode)
