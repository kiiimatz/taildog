import { useState } from 'react'
import { X } from 'lucide-react'

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

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <div className="w-[360px] rounded-2xl border border-gray-700 bg-gray-900 shadow-2xl">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-gray-800 px-5 py-4">
          <div className="flex items-center gap-2">
            <span className="text-gray-400">⚙</span>
            <h2 className="font-semibold text-white">Tunnel config</h2>
          </div>
          <button onClick={onCancel} className="text-gray-500 hover:text-white">
            <X size={16} />
          </button>
        </div>

        {/* Body */}
        <div className="px-5 py-4 space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-400 mb-1">Protocol</label>
            <select
              value={protocol}
              onChange={(e) => setProtocol(e.target.value)}
              className="w-full rounded-lg bg-gray-800 border border-gray-700 px-3 py-2 text-white focus:outline-none focus:ring-2 focus:ring-brand"
            >
              {PROTOCOLS.map((p) => (
                <option key={p} value={p}>{p.toUpperCase()}</option>
              ))}
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-400 mb-1">Client port</label>
            <input
              type="number"
              value={localPort}
              onChange={(e) => setLocalPort(e.target.value)}
              placeholder="e.g. 3000"
              min={1}
              max={65535}
              className="w-full rounded-lg bg-gray-800 border border-gray-700 px-3 py-2 text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-brand"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-400 mb-1">
              Server port{' '}
              <span className="text-xs text-gray-500">
                (optional — defaults to client port, auto-assigns if conflict)
              </span>
            </label>
            <input
              type="number"
              value={remotePort}
              onChange={(e) => setRemotePort(e.target.value)}
              placeholder={localPort || 'same as client port'}
              min={1}
              max={65535}
              className="w-full rounded-lg bg-gray-800 border border-gray-700 px-3 py-2 text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-brand"
            />
          </div>
        </div>

        {/* Footer */}
        <div className="flex justify-end gap-2 border-t border-gray-800 px-5 py-4">
          <button
            onClick={onCancel}
            className="rounded-lg px-4 py-2 text-sm text-gray-400 hover:text-white hover:bg-gray-800 transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={handleConnect}
            disabled={!localPort}
            className="rounded-lg bg-brand px-4 py-2 text-sm font-medium text-white hover:bg-blue-500 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Connect
          </button>
        </div>
      </div>
    </div>
  )
}
