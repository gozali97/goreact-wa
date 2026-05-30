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
import Login from './pages/Login.jsx'

async function loadFlyonUI() {
  return import('flyonui/flyonui')
}

export default function App() {
  const location = useLocation()
  const queryClient = useQueryClient()
  const { setDeviceStatus, setQrCode, setQrPaired } = useAppStore()
  const isAuthed = useAuthStore((s) => s.isAuthed)

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
    })

    const offQr = wsClient.on('qr', (data) => {
      setQrCode(data?.code || '')
      if (data?.paired) setQrPaired(true)
    })

    const offMsg = wsClient.on('message.new', () => {
      queryClient.invalidateQueries({ queryKey: ['conversations'] })
      queryClient.invalidateQueries({ queryKey: ['chat-detail'] })
    })

    const offBroadcast = wsClient.on('broadcast.update', () => {
      queryClient.invalidateQueries({ queryKey: ['broadcasts'] })
    })

    return () => {
      offStatus()
      offQr()
      offMsg()
      offBroadcast()
      wsClient.close()
    }
  }, [isAuthed, queryClient, setDeviceStatus, setQrCode, setQrPaired])

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
