// WebSocket event stream from relay
import { getAccessToken } from './api'
import { useAppStore } from '../store'

let ws: WebSocket | null = null
let reconnectTimer: ReturnType<typeof setTimeout> | null = null
let onOpenCallback: (() => void) | undefined

export function connectEvents(onOpen?: () => void) {
  if (onOpen !== undefined) onOpenCallback = onOpen

  const token = getAccessToken()
  if (!token) return

  const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws'
  const url = `${protocol}://${window.location.host}/api/events?token=${encodeURIComponent(token)}`

  ws = new WebSocket(url)

  ws.onopen = () => {
    console.log('[renode] WS connected to', url)
    // Re-fetch data on every (re)connect so we never miss events that fired
    // while the WebSocket was down.
    onOpenCallback?.()
  }

  ws.onmessage = (ev) => {
    console.log('[renode] WS message:', ev.data)
    try {
      const { type, data } = JSON.parse(ev.data)
      const store = useAppStore.getState()
      switch (type) {
        case 'CLIENT_CONNECTED':
          console.log('[renode] CLIENT_CONNECTED:', data)
          store.upsertClient({ ...data, online: true })
          break
        case 'CLIENT_DISCONNECTED':
          console.log('[renode] CLIENT_DISCONNECTED:', data)
          // Only flip the online flag — keep the existing client data intact.
          store.setClientOnline(data.id, false)
          break
        case 'TUNNEL_CREATED':
          store.addTunnel(data.clientID, data)
          break
        case 'TUNNEL_DELETED':
          store.removeTunnel(data.id)
          break
      }
    } catch {
      // ignore parse errors
    }
  }

  ws.onclose = (ev) => {
    console.log('[renode] WS closed — code:', ev.code, 'reason:', ev.reason)
    reconnectTimer = setTimeout(() => connectEvents(), 3000)
  }
  ws.onerror = (ev) => {
    console.error('[renode] WS error:', ev)
    ws?.close()
  }
}

export function disconnectEvents() {
  if (reconnectTimer) clearTimeout(reconnectTimer)
  if (ws) { ws.onclose = null; ws.close(); ws = null }
}
