import { useState } from 'react'

interface Props {
  clientID: string
  onConfirm: (proto: string, localPort: number, remotePort: number) => void
  onCancel: () => void
}

export default function TunnelConfigModal({ onConfirm, onCancel }: Props) {
  const [localPort, setLocalPort] = useState('')
  const [remotePort, setRemotePort] = useState('')

  function handleConnect() {
    const lp = parseInt(localPort, 10)
    const rp = remotePort ? parseInt(remotePort, 10) : lp
    if (!lp || lp < 1 || lp > 65535) return
    onConfirm('tcp', lp, rp)
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center"
      style={{ background: 'rgba(0,0,0,0.65)' }}
    >
      <div
        className="w-[400px] rounded-2xl"
        style={{
          background: '#1a1b28',
          border: '1px solid rgba(255,255,255,0.08)',
          boxShadow: '0 24px 64px rgba(0,0,0,0.6)',
        }}
      >
        {/* Header */}
        <div className="flex items-center gap-3 px-6 pt-6 pb-5">
          <span style={{ color: '#378ADD', fontSize: 18, lineHeight: 1 }}>⊙</span>
          <h2 className="font-semibold text-base" style={{ color: '#e8e9f0' }}>
            Tunnel config
          </h2>
        </div>

        {/* Body */}
        <div className="px-6 pb-5 space-y-5">
          {/* Client port */}
          <div>
            <label
              className="block text-sm font-medium mb-2"
              style={{ color: '#e8e9f0' }}
            >
              Client port
            </label>
            <input
              type="number"
              value={localPort}
              onChange={(e) => setLocalPort(e.target.value)}
              placeholder="3000"
              min={1}
              max={65535}
              className="w-full rounded-lg px-4 py-3 text-base focus:outline-none"
              style={{
                background: '#0f1018',
                border: '1px solid rgba(255,255,255,0.08)',
                color: '#e8e9f0',
              }}
              onFocus={(e) => {
                (e.currentTarget as HTMLInputElement).style.borderColor = 'rgba(55,138,221,0.5)'
              }}
              onBlur={(e) => {
                (e.currentTarget as HTMLInputElement).style.borderColor = 'rgba(255,255,255,0.08)'
              }}
            />
          </div>

          {/* Server port */}
          <div>
            <label
              className="block text-sm font-medium mb-2"
              style={{ color: '#e8e9f0' }}
            >
              Server port{' '}
              <span style={{ color: '#6b7280', fontWeight: 400 }}>(optional)</span>
            </label>
            <input
              type="number"
              value={remotePort}
              onChange={(e) => setRemotePort(e.target.value)}
              placeholder={localPort || '2000'}
              min={1}
              max={65535}
              className="w-full rounded-lg px-4 py-3 text-base focus:outline-none"
              style={{
                background: '#0f1018',
                border: '1px solid rgba(255,255,255,0.08)',
                color: '#e8e9f0',
              }}
              onFocus={(e) => {
                (e.currentTarget as HTMLInputElement).style.borderColor = 'rgba(55,138,221,0.5)'
              }}
              onBlur={(e) => {
                (e.currentTarget as HTMLInputElement).style.borderColor = 'rgba(255,255,255,0.08)'
              }}
            />
            <p className="text-xs mt-2 leading-relaxed" style={{ color: '#6b7280' }}>
              未入力の場合、サーバーポートはクライアントポートと同じになります。ポート衝突時は自動で別ポートが割り当てられます。
            </p>
          </div>
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between gap-3 px-6 pb-6">
          <button
            onClick={onCancel}
            className="flex-1 rounded-xl py-2.5 text-sm font-medium transition-colors"
            style={{
              background: '#242535',
              color: '#e8e9f0',
              border: '1px solid rgba(255,255,255,0.07)',
            }}
            onMouseEnter={(e) => {
              (e.currentTarget as HTMLButtonElement).style.background = '#2e2f42'
            }}
            onMouseLeave={(e) => {
              (e.currentTarget as HTMLButtonElement).style.background = '#242535'
            }}
          >
            Cancel
          </button>
          <button
            onClick={handleConnect}
            disabled={!localPort}
            className="flex-1 rounded-xl py-2.5 text-sm font-medium transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
            style={{
              background: '#e8e9f0',
              color: '#0c0d14',
            }}
            onMouseEnter={(e) => {
              if (localPort) {
                (e.currentTarget as HTMLButtonElement).style.background = '#ffffff'
              }
            }}
            onMouseLeave={(e) => {
              (e.currentTarget as HTMLButtonElement).style.background = '#e8e9f0'
            }}
          >
            Connect
          </button>
        </div>
      </div>
    </div>
  )
}
