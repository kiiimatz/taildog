import { useEffect } from 'react'
import { ReactFlowProvider } from '@xyflow/react'
import Sidebar from '../components/Sidebar'
import Canvas from '../components/canvas/Canvas'
import Toasts from '../components/Toasts'
import { getClients, getServerInfo } from '../lib/api'
import { connectEvents, disconnectEvents } from '../lib/ws'
import { useAppStore } from '../store'

export default function Dashboard() {
  const setClients = useAppStore((s) => s.setClients)
  const setServerInfo = useAppStore((s) => s.setServerInfo)

  useEffect(() => {
    async function fetchData() {
      try {
        const [clients, info] = await Promise.all([getClients(), getServerInfo()])
        setClients(clients)
        setServerInfo(info)
      } catch {
        // retry on WS reconnect
      }
    }
    fetchData()
    connectEvents()
    return () => disconnectEvents()
  }, [])

  return (
    <ReactFlowProvider>
      {/* Outer frame: viewport-sized, bordered, rounded, clipped */}
      <div style={{
        width: '100vw',
        height: '100vh',
        display: 'flex',
        border: '0.5px solid var(--border-primary)',
        borderRadius: 12,
        overflow: 'hidden',
      }}>
        <Sidebar />
        <Canvas />
      </div>
      <Toasts />
    </ReactFlowProvider>
  )
}
