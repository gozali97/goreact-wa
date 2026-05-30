import { useLocation } from 'react-router-dom'
import { useAppStore } from '../store/useAppStore.js'
import { useAuthStore } from '../store/useAuthStore.js'

const TITLES = {
  '/': { title: 'Hello there 👋', subtitle: 'Welcome back to your dashboard' },
  '/device': { title: 'Device', subtitle: 'Connect your WhatsApp account' },
  '/chats': { title: 'Chats', subtitle: 'Monitor and reply to conversations' },
  '/broadcast': { title: 'Broadcast', subtitle: 'Send a message to many numbers' },
  '/logs': { title: 'Activity Logs', subtitle: 'Audit every operation and event' },
  '/storage': { title: 'Storage', subtitle: 'Monitor disk usage and manage media files' },
  '/docs': { title: 'API Docs', subtitle: 'Integrate with external apps' },
  '/settings': { title: 'Settings', subtitle: 'Application configuration' },
}

export default function Header({ onMenu }) {
  const { pathname } = useLocation()
  const meta = TITLES[pathname] || { title: 'WA Proxy', subtitle: '' }
  const deviceStatus = useAppStore((s) => s.deviceStatus)
  const logout = useAuthStore((s) => s.logout)
  const connected = deviceStatus === 'connected'

  return (
    <header className="flex items-center gap-3 rounded-2xl bg-canvas-soft px-3 py-3 sm:px-4">
      {/* Mobile menu */}
      <button
        onClick={onMenu}
        className="flex h-10 w-10 flex-none items-center justify-center rounded-xl bg-white/5 text-white hover:bg-white/10 lg:hidden"
        aria-label="Open menu"
      >
        <span className="icon-[tabler--menu-2] size-5" />
      </button>

      {/* Greeting */}
      <div className="min-w-0 flex-1">
        <h2 className="truncate text-base font-bold text-white sm:text-lg">{meta.title}</h2>
        <p className="hidden truncate text-xs text-gray-400 sm:block">{meta.subtitle}</p>
      </div>

      {/* Search (hidden on small screens) */}
      <div className="relative hidden md:block md:w-64 lg:w-80">
        <span className="icon-[tabler--search] absolute left-3 top-1/2 size-4 -translate-y-1/2 text-gray-400" />
        <input
          type="text"
          placeholder="Search here..."
          className="w-full rounded-xl border border-white/10 bg-white/5 py-2.5 pl-9 pr-3 text-sm text-white placeholder:text-gray-500 focus:border-brand-500 focus:outline-none"
        />
      </div>

      {/* Actions */}
      <div className="flex flex-none items-center gap-2">
        <button
          className="relative flex h-10 w-10 items-center justify-center rounded-xl bg-white/5 text-white hover:bg-white/10"
          aria-label="Notifications"
        >
          <span className="icon-[tabler--bell] size-5" />
          {connected && (
            <span className="absolute right-2.5 top-2.5 h-2 w-2 rounded-full bg-brand-400" />
          )}
        </button>
        <button
          onClick={logout}
          className="flex h-10 w-10 items-center justify-center rounded-xl bg-white/5 text-white hover:bg-white/10"
          aria-label="Sign out"
          title="Sign out"
        >
          <span className="icon-[tabler--logout] size-5" />
        </button>
        <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-gradient-to-br from-brand-400 to-brand-600 text-sm font-semibold text-white">
          WA
        </div>
      </div>
    </header>
  )
}
