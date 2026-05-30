import { NavLink } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { contactApi } from '../lib/api.js'
import StatusBadge from './StatusBadge.jsx'
import { useAppStore } from '../store/useAppStore.js'

const NAV = [
  { to: '/', label: 'Dashboard', icon: 'icon-[tabler--layout-dashboard]', end: true },
  { to: '/device', label: 'Device', icon: 'icon-[tabler--device-mobile]' },
  { to: '/chats', label: 'Chats', icon: 'icon-[tabler--message-circle]' },
  { to: '/broadcast', label: 'Broadcast', icon: 'icon-[tabler--speakerphone]' },
  { to: '/logs', label: 'Logs', icon: 'icon-[tabler--file-text]' },
  { to: '/storage', label: 'Storage', icon: 'icon-[tabler--database]' },
  { to: '/docs', label: 'API Docs', icon: 'icon-[tabler--book-2]' },
  { to: '/settings', label: 'Settings', icon: 'icon-[tabler--settings]' },
]

function initials(text) {
  return (text || '?').trim().charAt(0).toUpperCase()
}

export default function Sidebar({ onNavigate }) {
  const deviceStatus = useAppStore((s) => s.deviceStatus)

  const { data } = useQuery({
    queryKey: ['contacts'],
    queryFn: () => contactApi.list(''),
    refetchInterval: 30000,
  })
  const recipients = (data?.contacts || []).slice(0, 6)

  return (
    <div className="flex h-full flex-col bg-white">
      {/* Brand */}
      <div className="flex items-center gap-3 px-5 py-5">
        <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-brand-500 text-white">
          <span className="icon-[tabler--brand-whatsapp] size-5" />
        </div>
        <div>
          <h1 className="text-base font-bold leading-tight text-gray-900">WA Proxy</h1>
          <p className="text-xs text-gray-400">Platform</p>
        </div>
      </div>

      {/* Nav */}
      <nav className="flex-1 space-y-1 overflow-y-auto px-3 pb-4">
        {NAV.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            end={item.end}
            onClick={onNavigate}
            className={({ isActive }) =>
              `flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium transition ${
                isActive
                  ? 'bg-brand-50 text-brand-700'
                  : 'text-gray-500 hover:bg-gray-50 hover:text-gray-900'
              }`
            }
          >
            <span className={`${item.icon} size-5`} />
            {item.label}
          </NavLink>
        ))}

        {/* Recipients */}
        <div className="pt-5">
          <p className="px-3 pb-2 text-[11px] font-semibold uppercase tracking-wider text-gray-400">
            Recipients
          </p>
          <div className="space-y-0.5">
            {recipients.length === 0 && (
              <p className="px-3 py-2 text-xs text-gray-400">No contacts yet.</p>
            )}
            {recipients.map((c) => (
              <NavLink
                key={c.id}
                to="/chats"
                onClick={onNavigate}
                className="flex items-center gap-3 rounded-xl px-3 py-2 text-sm text-gray-600 hover:bg-gray-50"
              >
                <span className="flex h-7 w-7 flex-none items-center justify-center rounded-full bg-gradient-to-br from-brand-400 to-brand-600 text-xs font-semibold text-white">
                  {initials(c.name || c.phone)}
                </span>
                <span className="truncate">{c.name || c.phone}</span>
              </NavLink>
            ))}
          </div>
        </div>
      </nav>

      {/* Status footer */}
      <div className="border-t border-gray-100 px-5 py-4">
        <StatusBadge status={deviceStatus} />
      </div>
    </div>
  )
}
