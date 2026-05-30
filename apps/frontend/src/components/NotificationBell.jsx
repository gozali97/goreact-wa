import { useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useNotifications } from '../store/useNotifications.js'

const ICONS = {
  message: 'icon-[tabler--message-2]',
  device: 'icon-[tabler--device-mobile]',
  broadcast: 'icon-[tabler--speakerphone]',
}

function timeAgo(ts) {
  const s = Math.floor((Date.now() - ts) / 1000)
  if (s < 60) return 'just now'
  const m = Math.floor(s / 60)
  if (m < 60) return `${m}m ago`
  const h = Math.floor(m / 60)
  if (h < 24) return `${h}h ago`
  return `${Math.floor(h / 24)}d ago`
}

export default function NotificationBell() {
  const navigate = useNavigate()
  const [open, setOpen] = useState(false)
  const ref = useRef(null)

  const items = useNotifications((s) => s.items)
  const unread = useNotifications((s) => s.unread)
  const markAllRead = useNotifications((s) => s.markAllRead)
  const clear = useNotifications((s) => s.clear)

  // Close on outside click.
  useEffect(() => {
    const handler = (e) => {
      if (ref.current && !ref.current.contains(e.target)) setOpen(false)
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [])

  const toggle = () => {
    const next = !open
    setOpen(next)
    if (next && unread > 0) markAllRead()
  }

  const openItem = (n) => {
    setOpen(false)
    if (n.to) navigate(n.to)
  }

  return (
    <div className="relative" ref={ref}>
      <button
        onClick={toggle}
        className="relative flex h-10 w-10 items-center justify-center rounded-xl bg-white/5 text-white hover:bg-white/10"
        aria-label="Notifications"
      >
        <span className="icon-[tabler--bell] size-5" />
        {unread > 0 && (
          <span className="absolute -right-1 -top-1 flex h-4 min-w-4 items-center justify-center rounded-full bg-red-500 px-1 text-[10px] font-bold text-white">
            {unread > 9 ? '9+' : unread}
          </span>
        )}
      </button>

      {open && (
        <div className="absolute right-0 z-50 mt-2 w-80 overflow-hidden rounded-2xl border border-gray-200 bg-white shadow-xl">
          <div className="flex items-center justify-between border-b border-gray-100 px-4 py-3">
            <h3 className="text-sm font-semibold text-gray-900">Notifications</h3>
            {items.length > 0 && (
              <button
                onClick={clear}
                className="text-xs font-medium text-gray-400 hover:text-gray-700"
              >
                Clear all
              </button>
            )}
          </div>

          <div className="max-h-96 overflow-y-auto">
            {items.length === 0 ? (
              <div className="flex flex-col items-center gap-2 px-4 py-10 text-center">
                <span className="icon-[tabler--bell-off] size-8 text-gray-300" />
                <p className="text-sm text-gray-400">No notifications yet.</p>
              </div>
            ) : (
              items.map((n) => (
                <button
                  key={n.id}
                  onClick={() => openItem(n)}
                  className="flex w-full items-start gap-3 border-b border-gray-50 px-4 py-3 text-left hover:bg-gray-50"
                >
                  <span className="mt-0.5 flex h-8 w-8 flex-none items-center justify-center rounded-lg bg-brand-50">
                    <span className={`${ICONS[n.type] || 'icon-[tabler--bell]'} size-4 text-brand-600`} />
                  </span>
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-sm font-medium text-gray-900">{n.title}</p>
                    {n.body && <p className="truncate text-xs text-gray-500">{n.body}</p>}
                    <p className="mt-0.5 text-[10px] text-gray-400">{timeAgo(n.time)}</p>
                  </div>
                  {!n.read && <span className="mt-1.5 h-2 w-2 flex-none rounded-full bg-brand-500" />}
                </button>
              ))
            )}
          </div>
        </div>
      )}
    </div>
  )
}
