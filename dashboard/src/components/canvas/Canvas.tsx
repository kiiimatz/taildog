import { useCallback, useRef, useState, useEffect } from 'react'
import { Maximize2 } from 'lucide-react'
import {
  ReactFlow,
  Background,
  BackgroundVariant,
  useNodesState,
  useEdgesState,
  type Node,
  type Edge,
  type Connection,
  type NodeTypes,
  MarkerType,
  useReactFlow,
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import ServerNode from './ServerNode'
import type { ServerNodeData } from './ServerNode'
import ClientNode from './ClientNode'
import type { ClientNodeData } from './ClientNode'
import TunnelNode from './TunnelNode'
import type { TunnelNodeData } from './TunnelNode'
import { useAppStore } from '../../store'
import type { CanvasClient } from '../../store'
import { createTunnel, deleteTunnel, getCanvasState, saveCanvasState, getAccessToken } from '../../lib/api'

const nodeTypes: NodeTypes = { server: ServerNode, client: ClientNode, tunnel: TunnelNode }
type RFNode = Node<Record<string, unknown>>

const DEFAULT_EDGE_STYLE = {
  type: 'default' as const,
  animated: true,
  style: { stroke: '#378ADD', strokeWidth: 1.5 },
  markerEnd: { type: MarkerType.Arrow, color: '#378ADD', width: 5, height: 5 },
}

export default function Canvas() {
  const [nodes, setNodes, onNodesChange] = useNodesState<RFNode>([])
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([])
  const [dragOver, setDragOver] = useState(false)
  const wrapper = useRef<HTMLDivElement>(null)
  const activatingRef = useRef<Set<string>>(new Set())
  const saveTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const loadedRef = useRef(false)       // prevents duplicate load
  const canSaveRef = useRef(false)      // only save after initial load completes
  const { screenToFlowPosition, getNodes, getEdges, fitView } = useReactFlow()

  const clients = useAppStore((s) => s.clients)
  const serverInfo = useAppStore((s) => s.serverInfo)
  const canvasClients = useAppStore((s) => s.canvasClients)
  const setCanvasClients = useAppStore((s) => s.setCanvasClients)
  const addCanvasClient = useAppStore((s) => s.addCanvasClient)
  const removeCanvasClient = useAppStore((s) => s.removeCanvasClient)
  const addToast = useAppStore((s) => s.addToast)
  const setIsDraggingCanvasNode = useAppStore((s) => s.setIsDraggingCanvasNode)

  // ── Stable callback refs ───────────────────────────────────────────────────
  const updateTunnelDataRef = useRef((_id: string, _field: string, _value: string) => {})
  const handleRemoveTunnelRef = useRef(async (_id: string) => {})
  const onActivateTunnelRef = useRef(async (_id: string) => {})
  const onDeactivateTunnelRef = useRef(async (_id: string) => {})

  updateTunnelDataRef.current = (nodeID: string, field: string, value: string) => {
    setNodes((nds) => nds.map((n) =>
      n.id === nodeID ? { ...n, data: { ...n.data, [field]: value } } : n
    ))
  }

  handleRemoveTunnelRef.current = async (nodeID: string) => {
    const currentNodes = getNodes()
    const tunnelNode = currentNodes.find((n) => n.id === nodeID)
    if (tunnelNode) {
      const d = tunnelNode.data as TunnelNodeData & Record<string, unknown>
      if (d.status === 'active' && d.tunnelID) {
        try { await deleteTunnel(d.tunnelID as string) } catch { /* ignore */ }
      }
    }
    activatingRef.current.delete(nodeID)
    setEdges((es) => es.filter((e) => e.source !== nodeID && e.target !== nodeID))
    setNodes((nds) => nds.filter((n) => n.id !== nodeID))
  }

  onActivateTunnelRef.current = async (nodeID: string) => {
    const currentNodes = getNodes()
    const currentEdges = getEdges()
    const tunnelNode = currentNodes.find((n) => n.id === nodeID)
    if (!tunnelNode) return

    const d = tunnelNode.data as TunnelNodeData & Record<string, unknown>
    if (!d.localPort) return

    const clientEdge = currentEdges.find((e) => e.target === nodeID)
    const serverEdge = currentEdges.find((e) => e.source === nodeID && e.target === 'server')
    if (!clientEdge || !serverEdge) {
      addToast('Connect client and server to this tunnel block first', 'info')
      return
    }

    const clientNode = currentNodes.find((n) => n.id === clientEdge.source)
    if (!clientNode) return
    const clientData = clientNode.data as ClientNodeData & Record<string, unknown>

    if (d.status === 'active' && d.tunnelID) {
      try { await deleteTunnel(d.tunnelID as string) } catch { /* ignore */ }
    }

    activatingRef.current.delete(nodeID)
    activatingRef.current.add(nodeID)

    setNodes((nds) => nds.map((n) =>
      n.id === nodeID ? { ...n, data: { ...n.data, status: 'connecting', tunnelID: undefined } } : n
    ))

    const lp = parseInt(d.localPort as string, 10)
    const rp = d.remotePort ? parseInt(d.remotePort as string, 10) : lp

    createTunnel({
      clientID: clientData.clientID as string,
      protocol: d.protocol as string,
      localPort: lp,
      remotePort: rp,
    })
      .then((result) => {
        setNodes((nds) => nds.map((n) =>
          n.id === nodeID
            ? { ...n, data: { ...n.data, status: 'active', tunnelID: result.id, remotePort: String(result.remotePort) } }
            : n
        ))
        if (result.remotePort !== rp) addToast(`Port ${rp} in use — assigned :${result.remotePort}`, 'info')
      })
      .catch((err) => {
        setNodes((nds) => nds.map((n) =>
          n.id === nodeID ? { ...n, data: { ...n.data, status: 'error' } } : n
        ))
        activatingRef.current.delete(nodeID)
        addToast(err instanceof Error ? err.message : 'Failed to create tunnel', 'error')
      })
  }

  onDeactivateTunnelRef.current = async (nodeID: string) => {
    const tunnelNode = getNodes().find((n) => n.id === nodeID)
    if (!tunnelNode) return
    const d = tunnelNode.data as TunnelNodeData & Record<string, unknown>
    if (d.tunnelID) {
      try { await deleteTunnel(d.tunnelID as string) } catch { /* ignore */ }
    }
    activatingRef.current.delete(nodeID)
    setNodes((nds) => nds.map((n) =>
      n.id === nodeID ? { ...n, data: { ...n.data, status: 'idle', tunnelID: undefined } } : n
    ))
  }

  const stableUpdateData = useCallback(
    (id: string, field: string, value: string) => updateTunnelDataRef.current(id, field, value), []
  )
  const stableRemoveTunnel = useCallback(
    async (id: string) => handleRemoveTunnelRef.current(id), []
  )
  const stableActivateTunnel = useCallback(
    async (id: string) => onActivateTunnelRef.current(id), []
  )
  const stableDeactivateTunnel = useCallback(
    async (id: string) => onDeactivateTunnelRef.current(id), []
  )

  function makeTunnelData(overrides: Partial<TunnelNodeData> = {}): TunnelNodeData & Record<string, unknown> {
    return {
      protocol: 'tcp', localPort: '', remotePort: '', status: 'idle',
      onDataChange: stableUpdateData, onRemove: stableRemoveTunnel,
      onActivate: stableActivateTunnel, onDeactivate: stableDeactivateTunnel,
      ...overrides,
    }
  }

  // ── Load canvas state once on mount ───────────────────────────────────────
  useEffect(() => {
    if (loadedRef.current) return
    loadedRef.current = true

    getCanvasState().then((saved) => {
      if (!saved || !Object.keys(saved).length) {
        canSaveRef.current = true
        return
      }

      // Restore canvas clients (positions)
      if (Array.isArray(saved.canvasClients) && (saved.canvasClients as CanvasClient[]).length > 0) {
        setCanvasClients(saved.canvasClients as CanvasClient[])
      }

      // Restore tunnel nodes — reconcile status against live server data
      if (Array.isArray(saved.tunnelNodes)) {
        type SavedTunnel = { id: string; posX: number; posY: number; protocol: string; localPort: string; remotePort: string; status?: string; tunnelID?: string }
        // Build set of tunnel IDs that are currently active on the server
        const { clients: currentClients } = useAppStore.getState()
        const activeTunnelIDs = new Set(currentClients.flatMap((c) => (c.tunnels ?? []).map((t) => t.id)))

        const tunnelNodes: RFNode[] = (saved.tunnelNodes as SavedTunnel[]).map((tn) => {
          const savedStatus = tn.status as TunnelNodeData['status'] | undefined
          // If clients already loaded, reconcile immediately; otherwise keep tunnelID
          // so the reconcile effect can restore active state once clients arrive.
          let status: TunnelNodeData['status'] = savedStatus ?? 'idle'
          if ((savedStatus === 'active' || savedStatus === 'connecting') && tn.tunnelID) {
            status = activeTunnelIDs.size > 0
              ? (activeTunnelIDs.has(tn.tunnelID) ? 'active' : 'idle')
              : 'idle'  // clients not loaded yet — will be restored by reconcile effect
          }
          return {
            id: tn.id,
            type: 'tunnel',
            position: { x: tn.posX, y: tn.posY },
            // Always preserve tunnelID so the reconcile effect can restore status later
            data: makeTunnelData({ protocol: tn.protocol, localPort: tn.localPort, remotePort: tn.remotePort, status, tunnelID: tn.tunnelID }),
          }
        })
        if (tunnelNodes.length > 0) {
          setNodes((nds) => [...nds, ...tunnelNodes])
        }
      }

      // Restore edges
      if (Array.isArray(saved.edges)) {
        type SavedEdge = { id: string; source: string; target: string }
        const restoredEdges: Edge[] = (saved.edges as SavedEdge[]).map((e) => ({
          ...DEFAULT_EDGE_STYLE,
          id: e.id,
          source: e.source,
          target: e.target,
        }))
        if (restoredEdges.length > 0) {
          setEdges(restoredEdges)
        }
      }
      // Allow saves only after load is complete
      canSaveRef.current = true
      // Fit all nodes into view after restoration
      setTimeout(() => fitView({ padding: 0.15, duration: 400 }), 100)
    }).catch(() => {
      canSaveRef.current = true
    })
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  // ── Build save payload from current React Flow state ─────────────────────
  const buildPayloadRef = useRef(() => ({} as Record<string, unknown>))
  buildPayloadRef.current = () => {
    const currentNodes = getNodes()
    const tunnelNodes = currentNodes
      .filter((n) => n.type === 'tunnel')
      .map((n) => {
        const d = n.data as TunnelNodeData & Record<string, unknown>
        return { id: n.id, posX: n.position.x, posY: n.position.y,
          protocol: d.protocol, localPort: d.localPort, remotePort: d.remotePort,
          status: d.status, tunnelID: d.tunnelID }
      })
    const { canvasClients: ccs } = useAppStore.getState()
    const savedCanvasClients = ccs.map((cc) => {
      const node = currentNodes.find((n) => n.id === `client-${cc.instanceID}`)
      return { instanceID: cc.instanceID, clientID: cc.clientID,
        posX: node?.position.x ?? cc.posX, posY: node?.position.y ?? cc.posY }
    })
    const savedEdges = getEdges().map((e) => ({ id: e.id, source: e.source, target: e.target }))
    return { canvasClients: savedCanvasClients, tunnelNodes, edges: savedEdges }
  }

  const saveNow = useCallback(() => {
    if (!canSaveRef.current) return
    if (saveTimerRef.current) { clearTimeout(saveTimerRef.current); saveTimerRef.current = null }
    saveCanvasState(buildPayloadRef.current()).catch(() => {})
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  // ── Debounced save on edge/tunnel-data changes ────────────────────────────
  useEffect(() => {
    if (!canSaveRef.current) return
    if (saveTimerRef.current) clearTimeout(saveTimerRef.current)
    saveTimerRef.current = setTimeout(() => {
      saveCanvasState(buildPayloadRef.current()).catch(() => {})
    }, 1200)
    return () => { if (saveTimerRef.current) clearTimeout(saveTimerRef.current) }
  }, [nodes, edges, canvasClients]) // eslint-disable-line react-hooks/exhaustive-deps

  // ── Save on page leave (tab close / navigate away) ────────────────────────
  useEffect(() => {
    const handleVisibility = () => { if (document.visibilityState === 'hidden') saveNow() }
    const handleUnload = () => {
      if (!canSaveRef.current) return
      const payload = buildPayloadRef.current()
      const token = getAccessToken()
      // keepalive allows the request to outlive the page
      fetch('/api/canvas', {
        method: 'PUT', keepalive: true,
        headers: { 'Content-Type': 'application/json', ...(token ? { Authorization: `Bearer ${token}` } : {}) },
        body: JSON.stringify(payload),
      }).catch(() => {})
    }
    document.addEventListener('visibilitychange', handleVisibility)
    window.addEventListener('beforeunload', handleUnload)
    return () => {
      document.removeEventListener('visibilitychange', handleVisibility)
      window.removeEventListener('beforeunload', handleUnload)
    }
  }, [saveNow]) // eslint-disable-line react-hooks/exhaustive-deps

  // ── Reconcile tunnel status against live server data ─────────────────────
  // Runs whenever clients updates; restores 'active' or resets stale tunnels to 'idle'
  useEffect(() => {
    if (!canSaveRef.current) return
    if (clients.length === 0) return
    const activeTunnelIDs = new Set(clients.flatMap((c) => (c.tunnels ?? []).map((t) => t.id)))
    setNodes((nds) => nds.map((n) => {
      if (n.type !== 'tunnel') return n
      const d = n.data as TunnelNodeData & Record<string, unknown>
      if (!d.tunnelID) return n
      const serverActive = activeTunnelIDs.has(d.tunnelID as string)
      if (serverActive && d.status !== 'active') {
        // Restore to active (e.g. after page reload when clients arrived after canvas)
        return { ...n, data: { ...n.data, status: 'active' } }
      }
      if (!serverActive && (d.status === 'active' || d.status === 'connecting')) {
        // Tunnel dropped on server side
        return { ...n, data: { ...n.data, status: 'idle', tunnelID: undefined } }
      }
      return n
    }))
  }, [clients]) // eslint-disable-line react-hooks/exhaustive-deps

  // ── Sync server node ──────────────────────────────────────────────────────
  useEffect(() => {
    setNodes((nds) => {
      const data: ServerNodeData & Record<string, unknown> = {
        name: serverInfo?.serverName ?? 'taildog-relay',
        ip: serverInfo?.serverIP ?? '',
        ports: [],
      }
      if (!nds.find((n) => n.id === 'server')) {
        return [...nds, { id: 'server', type: 'server', position: { x: 520, y: 240 }, data, deletable: false }]
      }
      return nds.map((n) => (n.id === 'server' ? { ...n, data } : n))
    })
  }, [serverInfo])

  // ── Sync canvas client nodes ──────────────────────────────────────────────
  useEffect(() => {
    const onRemove = async (instanceID: string) => {
      const clientNodeId = `client-${instanceID}`
      // Deactivate any tunnels connected to this client before removing
      const currentEdges = getEdges()
      const currentNodes = getNodes()
      const connectedTunnelIDs = currentEdges
        .filter((e) => e.source === clientNodeId && currentNodes.find((n) => n.id === e.target)?.type === 'tunnel')
        .map((e) => e.target)
      for (const tunnelID of connectedTunnelIDs) {
        await onDeactivateTunnelRef.current(tunnelID)
      }
      removeCanvasClient(instanceID)
      setEdges((es) => es.filter((e) => e.source !== clientNodeId))
    }
    setNodes((nds) => {
      const validNodeIDs = new Set(canvasClients.map((c) => `client-${c.instanceID}`))
      const kept = nds.filter((n) => n.id === 'server' || n.type === 'tunnel' || validNodeIDs.has(n.id))
      const existingIDs = new Set(kept.map((n) => n.id))
      const additions: RFNode[] = []

      for (const cc of canvasClients) {
        const nodeID = `client-${cc.instanceID}`
        const client = clients.find((c) => c.id === cc.clientID)
        const data: ClientNodeData & Record<string, unknown> = {
          instanceID: cc.instanceID,
          clientID: cc.clientID,
          name: client?.name ?? cc.clientID,
          ip: client?.ip ?? '',
          online: client?.online ?? false,
          ports: (client?.tunnels ?? []).map((t) => ({ port: t.localPort, protocol: t.protocol })),
          onRemove,
        }
        if (!existingIDs.has(nodeID)) {
          additions.push({ id: nodeID, type: 'client', position: { x: cc.posX, y: cc.posY }, data })
        } else {
          kept.forEach((n, i) => { if (n.id === nodeID) kept[i] = { ...n, data } })
        }
      }
      return [...kept, ...additions]
    })
  }, [canvasClients, clients])

  // ── Auto-center on window resize ─────────────────────────────────────────
  useEffect(() => {
    let timer: ReturnType<typeof setTimeout>
    const handleResize = () => {
      clearTimeout(timer)
      timer = setTimeout(() => fitView({ padding: 0.15, duration: 300 }), 150)
    }
    window.addEventListener('resize', handleResize)
    return () => { window.removeEventListener('resize', handleResize); clearTimeout(timer) }
  }, [fitView])

  // ── Drop handler ──────────────────────────────────────────────────────────
  const onDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    setDragOver(false)
    const bounds = wrapper.current?.getBoundingClientRect()
    if (!bounds) return
    const pos = screenToFlowPosition({ x: e.clientX - bounds.left, y: e.clientY - bounds.top })

    const clientID = e.dataTransfer.getData('clientID')
    const blockType = e.dataTransfer.getData('blockType')

    if (clientID) {
      const instanceID = Math.random().toString(36).slice(2)
      addCanvasClient({ instanceID, clientID, posX: pos.x, posY: pos.y })
    } else if (blockType === 'tunnel') {
      const nodeID = `tunnel-${Math.random().toString(36).slice(2)}`
      setNodes((nds) => [...nds, {
        id: nodeID,
        type: 'tunnel',
        position: pos,
        data: makeTunnelData(),
      }])
    }
  }, [screenToFlowPosition, addCanvasClient, setNodes])

  // ── Node drag: detect when dragging over sidebar ──────────────────────────
  const onNodeDrag = useCallback((event: React.MouseEvent) => {
    const canvasLeft = wrapper.current?.getBoundingClientRect().left ?? 0
    setIsDraggingCanvasNode(event.clientX < canvasLeft)
  }, [setIsDraggingCanvasNode])

  const onNodeDragStop = useCallback(async (event: React.MouseEvent, node: RFNode) => {
    const canvasLeft = wrapper.current?.getBoundingClientRect().left ?? 0
    setIsDraggingCanvasNode(false)

    if (event.clientX < canvasLeft) {
      // Dropped onto sidebar → delete the node (server is protected)
      if (node.id === 'server') return
      if (node.type === 'tunnel') {
        await handleRemoveTunnelRef.current(node.id)
      } else if (node.type === 'client') {
        // Deactivate any tunnels connected to this client before removing
        const currentEdges = getEdges()
        const currentNodes = getNodes()
        const connectedTunnelIDs = currentEdges
          .filter((e) => e.source === node.id && currentNodes.find((n) => n.id === e.target)?.type === 'tunnel')
          .map((e) => e.target)
        for (const tunnelID of connectedTunnelIDs) {
          await onDeactivateTunnelRef.current(tunnelID)
        }
        const instanceID = node.id.replace(/^client-/, '')
        removeCanvasClient(instanceID)
        setEdges((es) => es.filter((e) => e.source !== node.id))
      }
      addToast('Card removed', 'info')
    } else {
      saveNow()
    }
  }, [setIsDraggingCanvasNode, removeCanvasClient, setEdges, addToast, saveNow]) // eslint-disable-line react-hooks/exhaustive-deps

  // ── Connect handler ───────────────────────────────────────────────────────
  const onConnect = useCallback((connection: Connection) => {
    const currentNodes = getNodes()
    const currentEdges = getEdges()
    const sourceNode = currentNodes.find((n) => n.id === connection.source)
    const targetNode = currentNodes.find((n) => n.id === connection.target)
    if (!sourceNode || !targetNode) return

    const isClientToTunnel = sourceNode.type === 'client' && targetNode.type === 'tunnel'
    const isTunnelToServer = sourceNode.type === 'tunnel' && targetNode.id === 'server'
    if (!isClientToTunnel && !isTunnelToServer) return

    if (isClientToTunnel) {
      // Tunnel can only accept one client
      if (currentEdges.some((e) => e.target === connection.target)) {
        addToast('This tunnel already has a client', 'info')
        return
      }
    }

    if (isTunnelToServer) {
      // Tunnel can only connect to server once
      if (currentEdges.some((e) => e.source === connection.source && e.target === 'server')) {
        addToast('This tunnel is already connected to the server', 'info')
        return
      }
    }

    setEdges((es) => [...es, {
      ...DEFAULT_EDGE_STYLE,
      id: `e-${Date.now()}-${Math.random().toString(36).slice(2)}`,
      source: connection.source!,
      target: connection.target!,
    }])
  }, [getNodes, getEdges, setEdges, addToast])

  // ── Edge click ────────────────────────────────────────────────────────────
  async function handleEdgeClick(_: React.MouseEvent, edge: Edge) {
    const currentNodes = getNodes()
    const sourceNode = currentNodes.find((n) => n.id === edge.source)
    const targetNode = currentNodes.find((n) => n.id === edge.target)
    const tunnelNode = [sourceNode, targetNode].find((n) => n?.type === 'tunnel')

    if (!window.confirm('Remove this connection?')) return

    if (tunnelNode) {
      const d = tunnelNode.data as TunnelNodeData & Record<string, unknown>
      if (d.status === 'active' && d.tunnelID) {
        try { await deleteTunnel(d.tunnelID as string) } catch { /* ignore */ }
      }
      if (d.status === 'active' || d.status === 'connecting') {
        activatingRef.current.delete(tunnelNode.id)
        setNodes((nds) => nds.map((n) =>
          n.id === tunnelNode.id
            ? { ...n, data: { ...n.data, status: 'idle', tunnelID: undefined } }
            : n
        ))
      }
    }
    setEdges((es) => es.filter((e) => e.id !== edge.id))
  }

  return (
    <div
      ref={wrapper}
      style={{ flex: 1, position: 'relative' }}
      onDrop={onDrop}
      onDragOver={(e) => { e.preventDefault(); setDragOver(true) }}
      onDragLeave={() => setDragOver(false)}
    >
      {dragOver && (
        <div style={{
          position: 'absolute', inset: 8, zIndex: 10, pointerEvents: 'none',
          borderRadius: 12, border: '1.5px dashed rgba(55,138,221,0.4)',
          background: 'rgba(55,138,221,0.04)',
        }} />
      )}

      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onConnect={onConnect}
        onEdgeClick={handleEdgeClick}
        onNodeDrag={onNodeDrag}
        onNodeDragStop={onNodeDragStop}
        nodeTypes={nodeTypes}
        defaultViewport={{ x: 0, y: 0, zoom: 1.6 }}
        minZoom={0.3}
        maxZoom={3}
        deleteKeyCode={null}
        proOptions={{ hideAttribution: true }}
        style={{ background: 'var(--bg-tertiary)' }}
      >
        <Background
          variant={BackgroundVariant.Lines}
          gap={32}
          lineWidth={0.5}
          color="var(--dot-color)"
        />
      </ReactFlow>

      {/* Fit-view button */}
      <button
        onClick={() => fitView({ padding: 0.15, duration: 400 })}
        title="Fit all nodes to view"
        style={{
          position: 'absolute', bottom: 16, left: 16, zIndex: 10,
          width: 28, height: 28, borderRadius: 7,
          background: 'var(--bg-primary)',
          border: '0.5px solid var(--border-primary)',
          color: 'var(--text-tertiary)',
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          cursor: 'pointer',
          boxShadow: '0 1px 4px rgba(0,0,0,0.08)',
        }}
        onMouseEnter={(e) => {
          (e.currentTarget as HTMLButtonElement).style.color = 'var(--text-primary)'
          ;(e.currentTarget as HTMLButtonElement).style.borderColor = 'var(--border-primary)'
        }}
        onMouseLeave={(e) => {
          (e.currentTarget as HTMLButtonElement).style.color = 'var(--text-tertiary)'
        }}
      >
        <Maximize2 size={13} />
      </button>
    </div>
  )
}
