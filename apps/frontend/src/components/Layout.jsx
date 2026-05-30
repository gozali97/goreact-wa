import { useState } from 'react'
import { Outlet } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { deviceApi } from '../lib/api.js'
import { useAppStore } from '../store/useAppStore.js'
import Sidebar from './Sidebar.jsx'
import Header from './Header.jsx'

export default function Layout() {
  const [mobileOpen, setMobileOpen] = useState(false)
  const setDeviceStatus = useAppStore((s) => s.setDeviceStatus)

  // Poll status as a fallback to WebSocket.
  useQuery({
    queryKey: ['device-status'],
    queryFn: async () => {
      const data = await deviceApi.status()
      if (data?.status) setDeviceStatus(data.status)
      return data
    },
    refetchInterval: 15000,
  })

  return (
    <div className="flex h-full bg-white">
      {/* Desktop sidebar */}
      <aside className="hidden w-64 flex-none border-r border-gray-100 lg:block">
        <Sidebar />
      </aside>

      {/* Mobile drawer */}
      {mobileOpen && (
        <div className="fixed inset-0 z-40 lg:hidden">
          <div
            className="absolute inset-0 bg-black/40"
            onClick={() => setMobileOpen(false)}
          />
          <div className="absolute left-0 top-0 h-full w-72 max-w-[80%] shadow-xl">
            <Sidebar onNavigate={() => setMobileOpen(false)} />
          </div>
        </div>
      )}

      {/* Content canvas */}
      <main className="flex flex-1 flex-col overflow-hidden bg-canvas p-3 sm:p-4 lg:p-5">
        <Header onMenu={() => setMobileOpen(true)} />
        <div className="mt-3 flex-1 overflow-y-auto scrollbar-dark sm:mt-4">
          <Outlet />
        </div>
      </main>
    </div>
  )
}
