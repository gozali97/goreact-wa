import { useEffect } from 'react'
import { Routes, Route, Navigate, useLocation } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import Layout from './components/Layout.jsx'
import Dashboard from './pages/Dashboard.jsx'
import Device from './pages/Device.jsx'
import Chats from './pages/Chats.jsx'
import Broadcast from './pages/Broadcast.jsx'
import ApiDocs from './pages/ApiDocs.jsx'
import Settings from './pages/Settings.jsx'
import Logs from './pages/Logs.jsx'
import Storage from './pages/Storage.jsx'
import wsClient from './lib/ws.js'
import { useAppStore } from './store/useAppStore.js'
import { useAuthStore } from './store/useAuthStore.js'
import { useNotifications } from './store/useNotifications.js'
import Login from './pages/Login.jsx'

async function loadFlyonUI() {
  return import('flyonui/flyonui')
}

export default function App() {
  const location = useLocation()
  const queryClient = useQueryClient()
  const { setDeviceStatus, setQrCode, setQrPaired } = useAppStore()
  const isAuthed = useAuthStore((s) => s.isAuthed)
  const addNotification = useNotifications((s) => s.add)

  // Load FlyonUI interactive components once.
  useEffect(() => {
    loadFlyonUI()
  }, [])

  // Re-init FlyonUI components on route change.
  useEffect(() => {
    const t = setTimeout(() => {
      if (window.HSStaticMethods?.autoInit) {
        window.HSStaticMethods.autoInit()
      }
    }, 100)
    return () => clearTimeout(t)
  }, [location.pathname])

  useEffect(() => {
    if (!isAuthed) return

    wsClient.connect()

    const offStatus = wsClient.on('device.status', (data) => {
      if (data?.status) setDeviceStatus(data.status)
      queryClient.invalidateQueries({ queryKey: ['device-status'] })
      queryClient.invalidateQueries({ queryKey: ['dashboard-stats'] })
      const labels = {
        connected: 'WhatsApp connected',
        connecting: 'WhatsApp connecting…',
        disconnected: 'WhatsApp disconnected',
        logged_out: 'WhatsApp logged out',
      }
      if (data?.status && labels[data.status]) {
        addNotification({
          type: 'device',
          title: labels[data.status],
          body: '',
          to: '/device',
        })
      }
    })

    const offQr = wsClient.on('qr', (data) => {
      setQrCode(data?.code || '')
      if (data?.paired) setQrPaired(true)
    })

    const offMsg = wsClient.on('message.new', (data) => {
      queryClient.invalidateQueries({ queryKey: ['conversations'] })
      queryClient.invalidateQueries({ queryKey: ['chat-detail'] })
      // Only notify for incoming messages (not our own outgoing).
      if (data?.message?.direction === 'incoming') {
        const name = data.contact?.name || data.contact?.phone || 'New message'
        const m = data.message || {}
        const preview =
          m.content || (m.message_type ? `[${m.message_type}]` : 'New message')
        addNotification({
          type: 'message',
          title: name,
          body: preview,
          to: '/chats',
        })
      }
    })

    const offBroadcast = wsClient.on('broadcast.update', (data) => {
      queryClient.invalidateQueries({ queryKey: ['broadcasts'] })
      if (data?.status === 'success' || data?.status === 'failed') {
        addNotification({
          type: 'broadcast',
          title: `Broadcast ${data.status}`,
          body: data.name
            ? `${data.name} · ✓${data.total_success ?? 0} ✗${data.total_failed ?? 0}`
            : '',
          to: '/broadcast',
        })
      }
    })

    return () => {
      offStatus()
      offQr()
      offMsg()
      offBroadcast()
      wsClient.close()
    }
  }, [isAuthed, queryClient, setDeviceStatus, setQrCode, setQrPaired, addNotification])

  if (!isAuthed) {
    return <Login />
  }

  return (
    <Routes>
      <Route path="/" element={<Layout />}>
        <Route index element={<Dashboard />} />
        <Route path="device" element={<Device />} />
        <Route path="chats" element={<Chats />} />
        <Route path="broadcast" element={<Broadcast />} />
        <Route path="logs" element={<Logs />} />
        <Route path="storage" element={<Storage />} />
        <Route path="docs" element={<ApiDocs />} />
        <Route path="settings" element={<Settings />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Route>
    </Routes>
  )
}
