import { useAppStore } from '../store'

export default function Sidebar() {
  const clients = useAppStore((s) => s.clients)
  const canvasClients = useAppStore((s) => s.canvasClients)

  function onDragStart(e: React.DragEvent, clientID: string) {
    e.dataTransfer.setData('clientID', clientID)
    e.dataTransfer.effectAllowed = 'copy'
  }

  const onCanvasIDs = new Set(canvasClients.map((c) => c.clientID))

  return (
    <aside className="w-60 flex-shrink-0 bg-gray-950 border-r border-gray-800 flex flex-col">
      {/* Header */}
      <div className="px-4 py-4 border-b border-gray-800">
        <div className="flex items-center gap-2">
          <span className="text-2xl">🐕</span>
          <span className="font-bold text-white text-lg tracking-tight">taildog</span>
        </div>
      </div>

      {/* Connected clients */}
      <div className="flex-1 overflow-y-auto px-3 py-4">
        <div className="flex items-center justify-between mb-3">
          <span className="text-xs font-semibold uppercase tracking-widest text-gray-500">
            Connected clients
          </span>
        </div>

        {clients.length === 0 ? (
          <p className="text-xs text-gray-600 italic px-1">No clients connected</p>
        ) : (
          <div className="space-y-1">
            {clients.map((client) => {
              const isOnCanvas = onCanvasIDs.has(client.id)
              return (
                <div
                  key={client.id}
                  draggable={!isOnCanvas}
                  onDragStart={(e) => !isOnCanvas && onDragStart(e, client.id)}
                  className={`flex items-center gap-2.5 rounded-lg px-3 py-2 text-sm transition-colors ${
                    isOnCanvas
                      ? 'opacity-40 cursor-default'
                      : 'cursor-grab hover:bg-gray-800 active:cursor-grabbing'
                  }`}
                  title={isOnCanvas ? 'Already on canvas' : 'Drag to canvas'}
                >
                  <div
                    className={`w-2 h-2 rounded-full flex-shrink-0 ${
                      client.online ? 'bg-green-400 shadow-sm shadow-green-400' : 'bg-gray-600'
                    }`}
                  />
                  <div className="min-w-0">
                    <p className="font-medium text-gray-200 truncate">{client.name}</p>
                    <p className="text-xs text-gray-500 truncate">{client.ip}</p>
                  </div>
                </div>
              )
            })}
          </div>
        )}

        {/* Tunnels count */}
        <div className="mt-6 flex items-center justify-between">
          <span className="text-xs font-semibold uppercase tracking-widest text-gray-500">
            Tunnels
          </span>
          <span className="rounded-full bg-brand/20 text-brand text-xs font-mono px-2 py-0.5">
            {clients.reduce((acc, c) => acc + (c.tunnels?.length ?? 0), 0)}
          </span>
        </div>
      </div>

      {/* Footer hint */}
      <div className="px-4 py-3 border-t border-gray-800">
        <p className="text-xs text-gray-600 text-center">Drag clients to canvas</p>
      </div>
    </aside>
  )
}
