import { create } from 'zustand'

export interface TunnelInfo {
  id: string
  clientID: string
  protocol: string
  localPort: number
  remotePort: number
  active: boolean
  createdAt: string
}

export interface ClientInfo {
  id: string
  name: string
  ip: string
  connectedAt: string
  online: boolean
  tunnels: TunnelInfo[]
}

export interface ServerInfo {
  version: string
  uptime: number
  connectedClients: number
  serverName: string
  serverIP: string
}

export interface User {
  id: string
  username: string
  role: string
  lastLogin: string
}

// ── Toast ───────────────────────────────────────────────────────────────────
export interface Toast {
  id: string
  message: string
  type: 'info' | 'success' | 'error'
}

// ── Canvas node positions ────────────────────────────────────────────────────
export interface CanvasClient {
  instanceID: string  // unique per placement; same clientID can appear multiple times
  clientID: string
  posX: number
  posY: number
}

interface AppState {
  // Auth
  user: User | null
  setUser: (u: User | null) => void

  // Data
  clients: ClientInfo[]
  serverInfo: ServerInfo | null
  setClients: (clients: ClientInfo[]) => void
  setServerInfo: (info: ServerInfo) => void
  upsertClient: (client: ClientInfo) => void
  setClientOnline: (id: string, online: boolean) => void
  removeClient: (id: string) => void
  addTunnel: (clientID: string, tunnel: TunnelInfo) => void
  removeTunnel: (tunnelID: string) => void

  // Canvas
  canvasClients: CanvasClient[]
  setCanvasClients: (ccs: CanvasClient[]) => void
  addCanvasClient: (cc: CanvasClient) => void
  removeCanvasClient: (instanceID: string) => void
  updateCanvasClientPos: (instanceID: string, posX: number, posY: number) => void

  // Drag state
  isDraggingCanvasNode: boolean
  setIsDraggingCanvasNode: (v: boolean) => void

  // Toasts
  toasts: Toast[]
  addToast: (msg: string, type?: Toast['type']) => void
  removeToast: (id: string) => void
}

export const useAppStore = create<AppState>((set) => ({
  user: null,
  setUser: (user) => set({ user }),

  clients: [],
  serverInfo: null,
  setClients: (clients) => set({ clients }),
  setServerInfo: (serverInfo) => set({ serverInfo }),
  upsertClient: (client) =>
    set((s) => {
      const idx = s.clients.findIndex((c) => c.id === client.id)
      if (idx >= 0) {
        const clients = [...s.clients]
        clients[idx] = client
        return { clients }
      }
      return { clients: [...s.clients, client] }
    }),
  setClientOnline: (id, online) =>
    set((s) => ({
      clients: s.clients.map((c) => c.id === id ? { ...c, online } : c),
    })),
  removeClient: (id) =>
    set((s) => ({ clients: s.clients.filter((c) => c.id !== id) })),
  addTunnel: (clientID, tunnel) =>
    set((s) => ({
      clients: s.clients.map((c) =>
        c.id === clientID ? { ...c, tunnels: [...c.tunnels, tunnel] } : c
      ),
    })),
  removeTunnel: (tunnelID) =>
    set((s) => ({
      clients: s.clients.map((c) => ({
        ...c,
        tunnels: c.tunnels.filter((t) => t.id !== tunnelID),
      })),
    })),

  canvasClients: [],
  setCanvasClients: (ccs) => set({ canvasClients: ccs }),
  addCanvasClient: (cc) =>
    set((s) => ({ canvasClients: [...s.canvasClients, cc] })),
  removeCanvasClient: (instanceID) =>
    set((s) => ({ canvasClients: s.canvasClients.filter((c) => c.instanceID !== instanceID) })),
  updateCanvasClientPos: (instanceID, posX, posY) =>
    set((s) => ({
      canvasClients: s.canvasClients.map((c) =>
        c.instanceID === instanceID ? { ...c, posX, posY } : c
      ),
    })),

  isDraggingCanvasNode: false,
  setIsDraggingCanvasNode: (v) => set({ isDraggingCanvasNode: v }),

  toasts: [],
  addToast: (message, type = 'info') => {
    const id = Math.random().toString(36).slice(2)
    set((s) => ({ toasts: [...s.toasts, { id, message, type }] }))
    setTimeout(() => {
      set((s) => ({ toasts: s.toasts.filter((t) => t.id !== id) }))
    }, 4000)
  },
  removeToast: (id) => set((s) => ({ toasts: s.toasts.filter((t) => t.id !== id) })),
}))
