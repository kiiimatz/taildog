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
        // Will retry on WS reconnect
      }
    }
    fetchData()
    connectEvents()
    return () => disconnectEvents()
  }, [])

  return (
    <ReactFlowProvider>
      <div style={{ display: 'flex', height: '100vh', overflow: 'hidden' }}>
        <Sidebar />
        <Canvas />
      </div>
      <Toasts />
    </ReactFlowProvider>
  )
}
