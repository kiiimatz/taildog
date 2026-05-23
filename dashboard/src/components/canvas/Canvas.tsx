import { useCallback, useRef, useState, useEffect } from 'react'
import {
  ReactFlow,
  Background,
  BackgroundVariant,
  useNodesState,
  useEdgesState,
  addEdge,
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
import TunnelConfigModal from './TunnelConfigModal'
import { useAppStore } from '../../store'
import { createTunnel, deleteTunnel } from '../../lib/api'

const nodeTypes: NodeTypes = { server: ServerNode, client: ClientNode }
type RFNode = Node<Record<string, unknown>>

interface PendingConnection {
  sourceNodeId: string
  clientID: string
}

export default function Canvas() {
  const [nodes, setNodes, onNodesChange] = useNodesState<RFNode>([])
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([])
  const [pending, setPending] = useState<PendingConnection | null>(null)
  const [dragOver, setDragOver] = useState(false)
  const wrapper = useRef<HTMLDivElement>(null)
  const { screenToFlowPosition } = useReactFlow()

  const clients = useAppStore((s) => s.clients)
  const serverInfo = useAppStore((s) => s.serverInfo)
  const canvasClients = useAppStore((s) => s.canvasClients)
  const addCanvasClient = useAppStore((s) => s.addCanvasClient)
  const removeCanvasClient = useAppStore((s) => s.removeCanvasClient)
  const addToast = useAppStore((s) => s.addToast)

  // Sync server node
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

  // Sync canvas client nodes
  useEffect(() => {
    const onRemove = (id: string) => {
      removeCanvasClient(id)
      setEdges((es) => es.filter((e) => e.source !== `client-${id}`))
    }
    setNodes((nds) => {
      const validIDs = new Set(canvasClients.map((c) => `client-${c.clientID}`))
      const kept = nds.filter((n) => n.id === 'server' || validIDs.has(n.id))
      const existingIDs = new Set(kept.map((n) => n.id))
      const additions: RFNode[] = []
      for (const cc of canvasClients) {
        const nodeID = `client-${cc.clientID}`
        const client = clients.find((c) => c.id === cc.clientID)
        const data: ClientNodeData & Record<string, unknown> = {
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
          return kept.map((n) => n.id === nodeID ? { ...n, data } : n).concat(additions)
        }
      }
      return [...kept, ...additions]
    })
  }, [canvasClients, clients])

  const onDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    setDragOver(false)
    const clientID = e.dataTransfer.getData('clientID')
    if (!clientID || canvasClients.find((c) => c.clientID === clientID)) return
    const bounds = wrapper.current?.getBoundingClientRect()
    if (!bounds) return
    const pos = screenToFlowPosition({ x: e.clientX - bounds.left, y: e.clientY - bounds.top })
    addCanvasClient({ clientID, posX: pos.x, posY: pos.y })
  }, [canvasClients, screenToFlowPosition, addCanvasClient])

  const onConnect = useCallback((connection: Connection) => {
    if (connection.target !== 'server') return
    const clientID = (connection.source ?? '').replace('client-', '')
    setPending({ sourceNodeId: connection.source ?? '', clientID })
  }, [])

  async function handleTunnelConfirm(proto: string, localPort: number, remotePort: number) {
    if (!pending) return
    setPending(null)
    try {
      const result = await createTunnel({
        clientID: pending.clientID,
        protocol: proto,
        localPort,
        remotePort,
      })
      const assignedPort: number = result.remotePort
      if (assignedPort !== remotePort) {
        addToast(`Port ${remotePort} in use — assigned :${assignedPort} instead`, 'info')
      }
      setEdges((es) => addEdge({
        id: result.id,
        source: pending.sourceNodeId,
        target: 'server',
        type: 'smoothstep',
        animated: false,
        style: { stroke: '#378ADD', strokeWidth: 1.5, opacity: 0.7 },
        label: `:${localPort}→:${assignedPort}`,
        labelStyle: { fontSize: 9, fill: 'var(--text-tertiary)' },
        labelBgStyle: { fill: 'var(--bg-tertiary)', fillOpacity: 0.9 },
        markerEnd: { type: MarkerType.Arrow, color: '#378ADD', width: 6, height: 6 },
        data: { tunnelID: result.id, protocol: proto, localPort, remotePort: assignedPort },
      }, es))
    } catch (err: unknown) {
      addToast(err instanceof Error ? err.message : 'Failed to create tunnel', 'error')
    }
  }

  async function handleEdgeClick(_: React.MouseEvent, edge: Edge) {
    if (!window.confirm(`Delete tunnel ${edge.label ?? edge.id}?`)) return
    try {
      await deleteTunnel((edge.data as { tunnelID?: string })?.tunnelID ?? edge.id)
      setEdges((es) => es.filter((e) => e.id !== edge.id))
    } catch (err: unknown) {
      addToast(err instanceof Error ? err.message : 'Failed to delete tunnel', 'error')
    }
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
        nodeTypes={nodeTypes}
        fitView
        deleteKeyCode={null}
        style={{ background: 'var(--bg-tertiary)' }}
      >
        <Background
          variant={BackgroundVariant.Dots}
          gap={24}
          size={1}
          color="rgba(0,0,0,0.15)"
        />
      </ReactFlow>

      {pending && (
        <TunnelConfigModal
          clientID={pending.clientID}
          onConfirm={handleTunnelConfirm}
          onCancel={() => setPending(null)}
        />
      )}
    </div>
  )
}
