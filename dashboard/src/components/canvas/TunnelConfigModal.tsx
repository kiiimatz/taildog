import { useState } from 'react'
import { Settings2 } from 'lucide-react'

const PROTOCOLS = ['tcp', 'udp', 'http', 'https', 'socks5', 'quic', 'ws', 'scp', 'smtp']

interface Props {
  clientID: string
  onConfirm: (proto: string, localPort: number, remotePort: number) => void
  onCancel: () => void
}

export default function TunnelConfigModal({ onConfirm, onCancel }: Props) {
  const [protocol, setProtocol] = useState('tcp')
  const [localPort, setLocalPort] = useState('')
  const [remotePort, setRemotePort] = useState('')

  function handleConnect() {
    const lp = parseInt(localPort, 10)
    const rp = remotePort ? parseInt(remotePort, 10) : lp
    if (!lp || lp < 1 || lp > 65535) return
    onConfirm(protocol, lp, rp)
  }

  const inputStyle: React.CSSProperties = {
    width: '100%',
    fontSize: 12,
    padding: '5px 8px',
    border: '0.5px solid var(--border-secondary)',
    borderRadius: 8,
    background: 'var(--bg-secondary)',
    color: 'var(--text-primary)',
    outline: 'none',
  }

  return (
    <div style={{
      position: 'fixed', inset: 0, zIndex: 50,
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      background: 'rgba(0,0,0,0.18)',
    }}>
      <div style={{
        width: 240,
        background: 'var(--bg-primary)',
        border: '0.5px solid var(--border-secondary)',
        borderRadius: 12,
        padding: '18px 20px',
      }}>
        {/* Header */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 14 }}>
          <Settings2 size={14} color="#378ADD" />
          <span style={{ fontSize: 13, fontWeight: 500, color: 'var(--text-primary)' }}>
            Tunnel config
          </span>
        </div>

        {/* Protocol */}
        <div style={{ marginBottom: 10 }}>
          <label style={{ display: 'block', fontSize: 11, color: 'var(--text-secondary)', marginBottom: 4 }}>
            Protocol
          </label>
          <select
            value={protocol}
            onChange={(e) => setProtocol(e.target.value)}
            style={inputStyle}
          >
            {PROTOCOLS.map((p) => (
              <option key={p} value={p}>{p}</option>
            ))}
          </select>
        </div>

        {/* Client port */}
        <div style={{ marginBottom: 10 }}>
          <label style={{ display: 'block', fontSize: 11, color: 'var(--text-secondary)', marginBottom: 4 }}>
            Client port
          </label>
          <input
            type="number"
            value={localPort}
            onChange={(e) => setLocalPort(e.target.value)}
            placeholder="3000"
            min={1}
            max={65535}
            style={inputStyle}
            onFocus={(e) => { (e.currentTarget as HTMLInputElement).style.borderColor = '#378ADD' }}
            onBlur={(e) => { (e.currentTarget as HTMLInputElement).style.borderColor = 'var(--border-secondary)' }}
          />
        </div>

        {/* Server port */}
        <div style={{ marginBottom: 10 }}>
          <label style={{ display: 'block', fontSize: 11, color: 'var(--text-secondary)', marginBottom: 4 }}>
            Server port{' '}
            <span style={{ color: 'var(--text-tertiary)', fontWeight: 400 }}>(optional)</span>
          </label>
          <input
            type="number"
            value={remotePort}
            onChange={(e) => setRemotePort(e.target.value)}
            placeholder="same as client port"
            min={1}
            max={65535}
            style={inputStyle}
            onFocus={(e) => { (e.currentTarget as HTMLInputElement).style.borderColor = '#378ADD' }}
            onBlur={(e) => { (e.currentTarget as HTMLInputElement).style.borderColor = 'var(--border-secondary)' }}
          />
        </div>

        {/* Hint */}
        <p style={{ fontSize: 10, color: 'var(--text-tertiary)', lineHeight: 1.5, marginBottom: 12 }}>
          未入力の場合、サーバーポートはクライアントポートと同じになります。ポート衝突時は自動で別ポートが割り当てられます。
        </p>

        {/* Buttons */}
        <div style={{ display: 'flex', gap: 6, justifyContent: 'flex-end' }}>
          <button
            onClick={onCancel}
            style={{
              fontSize: 11, padding: '5px 12px',
              borderRadius: 8,
              border: '0.5px solid var(--border-secondary)',
              background: 'transparent',
              color: 'var(--text-primary)',
              cursor: 'pointer',
            }}
            onMouseEnter={(e) => { (e.currentTarget as HTMLButtonElement).style.background = 'var(--bg-secondary)' }}
            onMouseLeave={(e) => { (e.currentTarget as HTMLButtonElement).style.background = 'transparent' }}
          >
            Cancel
          </button>
          <button
            onClick={handleConnect}
            disabled={!localPort}
            style={{
              fontSize: 11, padding: '5px 12px',
              borderRadius: 8,
              border: '0.5px solid #185FA5',
              background: '#185FA5',
              color: '#ffffff',
              cursor: localPort ? 'pointer' : 'not-allowed',
              opacity: localPort ? 1 : 0.4,
            }}
            onMouseEnter={(e) => {
              if (localPort) (e.currentTarget as HTMLButtonElement).style.background = '#1a6bbf'
            }}
            onMouseLeave={(e) => {
              (e.currentTarget as HTMLButtonElement).style.background = '#185FA5'
            }}
          >
            Connect
          </button>
        </div>
      </div>
    </div>
  )
}
