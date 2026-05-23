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
import { MoreHorizontal } from 'lucide-react'
import ServerNode from './ServerNode'
import type { ServerNodeData } from './ServerNode'
import ClientNode from './ClientNode'
import type { ClientNodeData } from './ClientNode'
import TunnelConfigModal from './TunnelConfigModal'
import { useAppStore } from '../../store'
import { createTunnel, deleteTunnel } from '../../lib/api'

const nodeTypes: NodeTypes = {
  server: ServerNode,
  client: ClientNode,
}

// React Flow requires node data to be Record<string, unknown>
type RFNode = Node<Record<string, unknown>>

interface PendingConnection {
  sourceNodeId: string
  clientID: string
}

export default function Canvas() {
  const { fitView } = useReactFlow()
  const [nodes, setNodes, onNodesChange] = useNodesState<RFNode>([])
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([])
  const [pending, setPending] = useState<PendingConnection | null>(null)
  const [dragOver, setDragOver] = useState(false)
  const reactFlowWrapper = useRef<HTMLDivElement>(null)
  const { screenToFlowPosition } = useReactFlow()

  const clients = useAppStore((s) => s.clients)
  const serverInfo = useAppStore((s) => s.serverInfo)
  const canvasClients = useAppStore((s) => s.canvasClients)
  const addCanvasClient = useAppStore((s) => s.addCanvasClient)
  const removeCanvasClient = useAppStore((s) => s.removeCanvasClient)
  const addToast = useAppStore((s) => s.addToast)

  // Ensure server node exists
  useEffect(() => {
    setNodes((nds) => {
      const serverExists = nds.find((n) => n.id === 'server')
      const serverData: ServerNodeData & Record<string, unknown> = {
        name: serverInfo?.serverName ?? 'relay',
        ip: serverInfo?.serverIP ?? '',
        ports: [],
      }
      if (!serverExists) {
        return [
          ...nds,
          {
            id: 'server',
            type: 'server',
            position: { x: 600, y: 300 },
            data: serverData,
            deletable: false,
          },
        ]
      }
      return nds.map((n) => (n.id === 'server' ? { ...n, data: serverData } : n))
    })
  }, [serverInfo])

  // Sync canvas clients with store
  useEffect(() => {
    const onRemove = (id: string) => {
      removeCanvasClient(id)
      setEdges((es) => es.filter((e) => e.source !== `client-${id}`))
    }

    setNodes((nds) => {
      const validIDs = new Set(canvasClients.map((c) => `client-${c.clientID}`))
      const filtered = nds.filter((n) => n.id === 'server' || validIDs.has(n.id))

      const existingIDs = new Set(filtered.map((n) => n.id))
      const additions: RFNode[] = []

      for (const cc of canvasClients) {
        const nodeID = `client-${cc.clientID}`
        const client = clients.find((c) => c.id === cc.clientID)
        const nodeData: ClientNodeData & Record<string, unknown> = {
          clientID: cc.clientID,
          name: client?.name ?? cc.clientID,
          ip: client?.ip ?? '',
          online: client?.online ?? false,
          ports: (client?.tunnels ?? []).map((t) => ({ port: t.localPort, protocol: t.protocol })),
          onRemove,
        }
        if (!existingIDs.has(nodeID)) {
          additions.push({
            id: nodeID,
            type: 'client',
            position: { x: cc.posX, y: cc.posY },
            data: nodeData,
          })
        } else {
          return filtered.map((n) =>
            n.id === nodeID ? { ...n, data: nodeData } : n
          ).concat(additions)
        }
      }
      return [...filtered, ...additions]
    })
  }, [canvasClients, clients])

  // Handle drop from sidebar
  const onDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault()
      setDragOver(false)
      const clientID = e.dataTransfer.getData('clientID')
      if (!clientID) return
      if (canvasClients.find((c) => c.clientID === clientID)) return

      const bounds = reactFlowWrapper.current?.getBoundingClientRect()
      if (!bounds) return
      const pos = screenToFlowPosition({ x: e.clientX - bounds.left, y: e.clientY - bounds.top })
      addCanvasClient({ clientID, posX: pos.x, posY: pos.y })
    },
    [canvasClients, screenToFlowPosition, addCanvasClient]
  )

  const onConnect = useCallback(
    (connection: Connection) => {
      if (connection.target !== 'server') return
      const clientID = (connection.source ?? '').replace('client-', '')
      setPending({ sourceNodeId: connection.source ?? '', clientID })
    },
    []
  )

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
      const assignedPort = result.remotePort
      if (assignedPort !== remotePort) {
        addToast(`Port ${remotePort} in use — assigned :${assignedPort} instead`, 'info')
      }
      const edgeID = result.id
      setEdges((es) =>
        addEdge(
          {
            id: edgeID,
            source: pending.sourceNodeId,
            target: 'server',
            type: 'smoothstep',
            animated: true,
            style: { stroke: '#378ADD', strokeWidth: 1.5 },
            label: `:${localPort} → :${assignedPort}`,
            labelStyle: { fill: '#6b7280', fontSize: 11 },
            labelBgStyle: { fill: '#161622' },
            markerEnd: { type: MarkerType.ArrowClosed, color: '#378ADD' },
            data: { tunnelID: edgeID, protocol: proto, localPort, remotePort: assignedPort },
          },
          es
        )
      )
    } catch (err: unknown) {
      addToast(err instanceof Error ? err.message : 'Failed to create tunnel', 'error')
    }
  }

  async function handleEdgeClick(_: React.MouseEvent, edge: Edge) {
    if (!window.confirm(`Delete tunnel ${edge.label ?? edge.id}?`)) return
    try {
      const tunnelID = (edge.data as { tunnelID?: string })?.tunnelID ?? edge.id
      await deleteTunnel(tunnelID)
      setEdges((es) => es.filter((e) => e.id !== edge.id))
    } catch (err: unknown) {
      addToast(err instanceof Error ? err.message : 'Failed to delete tunnel', 'error')
    }
  }

  return (
    <div
      ref={reactFlowWrapper}
      className="flex-1 relative"
      onDrop={onDrop}
      onDragOver={(e) => { e.preventDefault(); setDragOver(true) }}
      onDragLeave={() => setDragOver(false)}
    >
      {dragOver && (
        <div
          className="absolute inset-0 z-10 pointer-events-none m-2 rounded-xl"
          style={{
            border: '2px dashed rgba(55,138,221,0.4)',
            background: 'rgba(55,138,221,0.04)',
          }}
        />
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
        defaultViewport={{ x: 0, y: 0, zoom: 1 }}
        deleteKeyCode={null}
        style={{ background: '#0c0d14' }}
      >
        <Background
          variant={BackgroundVariant.Dots}
          gap={24}
          size={1}
          color="rgba(255,255,255,0.06)"
        />

        {/* Top-right menu button */}
        <div className="absolute top-4 right-4 z-10">
          <button
            onClick={() => fitView({ padding: 0.2, duration: 500 })}
            className="flex items-center justify-center w-8 h-8 rounded-lg transition-colors"
            style={{
              background: '#161622',
              border: '1px solid rgba(255,255,255,0.08)',
              color: '#6b7280',
            }}
            onMouseEnter={(e) => {
              (e.currentTarget as HTMLButtonElement).style.color = '#e8e9f0'
              ;(e.currentTarget as HTMLButtonElement).style.borderColor = 'rgba(255,255,255,0.15)'
            }}
            onMouseLeave={(e) => {
              (e.currentTarget as HTMLButtonElement).style.color = '#6b7280'
              ;(e.currentTarget as HTMLButtonElement).style.borderColor = 'rgba(255,255,255,0.08)'
            }}
            title="Fit view"
          >
            <MoreHorizontal size={15} />
          </button>
        </div>
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
