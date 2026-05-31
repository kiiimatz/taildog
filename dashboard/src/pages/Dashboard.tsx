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
        console.log('[renode] getClients() returned:', clients)
        setClients(clients)
        setServerInfo(info)
      } catch (err) {
        console.error('[renode] fetchData() failed:', err)
      }
    }
    // Pass fetchData as the onOpen callback so we re-fetch the full client/server
    // list every time the WebSocket (re)connects.  This guarantees we never show
    // stale data when a CLIENT_CONNECTED event fired before the socket was ready.
    connectEvents(fetchData)
    return () => disconnectEvents()
  }, [])

  return (
    <ReactFlowProvider>
      <div style={{ width: '100vw', height: '100vh', position: 'relative', overflow: 'hidden' }}>
        {/* Canvas fills the entire viewport */}
        <Canvas />
        {/* Sidebar floats on top */}
        <Sidebar />
      </div>
      <Toasts />
    </ReactFlowProvider>
  )
}
