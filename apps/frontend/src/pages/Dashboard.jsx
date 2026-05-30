import { useQuery } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { dashboardApi } from '../lib/api.js'
import Card from '../components/Card.jsx'
import StatusBadge from '../components/StatusBadge.jsx'

function FeatureCard({ icon, iconBg, title, subtitle, value, valueLabel, meta, metaLabel }) {
  return (
    <Card className="p-5">
      <div className="flex items-center gap-3">
        <div className={`flex h-11 w-11 items-center justify-center rounded-xl ${iconBg}`}>
          <span className={`${icon} size-6`} />
        </div>
        <div className="min-w-0">
          <p className="truncate font-semibold text-gray-900">{title}</p>
          <p className="truncate text-xs text-gray-400">{subtitle}</p>
        </div>
      </div>
      <div className="mt-5 flex items-end justify-between">
        <div>
          <p className="text-2xl font-bold text-gray-900">{value}</p>
          <p className="text-xs text-gray-400">{valueLabel}</p>
        </div>
        <div className="text-right">
          <p className="text-sm font-semibold text-gray-700">{meta}</p>
          <p className="text-xs text-gray-400">{metaLabel}</p>
        </div>
      </div>
    </Card>
  )
}

export default function Dashboard() {
  const navigate = useNavigate()
  const { data, isLoading } = useQuery({
    queryKey: ['dashboard-stats'],
    queryFn: dashboardApi.stats,
    refetchInterval: 20000,
  })

  const device = data?.device || {}

  return (
    <div className="space-y-4 sm:space-y-5">
      {/* Feature cards */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
        <FeatureCard
          icon="icon-[tabler--users-group] text-blue-600"
          iconBg="bg-blue-50"
          title="Contacts"
          subtitle="People you've reached"
          value={data?.total_contacts ?? '—'}
          valueLabel="Total contacts"
          meta={device.connected ? 'Live' : 'Idle'}
          metaLabel="Sync"
        />
        <FeatureCard
          icon="icon-[tabler--message-2] text-brand-600"
          iconBg="bg-brand-50"
          title="Conversations"
          subtitle="Active chat threads"
          value={data?.total_chats ?? '—'}
          valueLabel="Total chats"
          meta="Inbox"
          metaLabel="Monitoring"
        />
        <FeatureCard
          icon="icon-[tabler--speakerphone] text-amber-600"
          iconBg="bg-amber-50"
          title="Broadcasts"
          subtitle="Bulk campaigns"
          value={data?.total_broadcasts ?? '—'}
          valueLabel="Total broadcasts"
          meta="Queue"
          metaLabel="Worker"
        />
      </div>

      {/* Main row */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        {/* Device overview (large) */}
        <Card className="p-5 lg:col-span-2">
          <div className="flex items-start justify-between">
            <div>
              <p className="text-sm text-gray-400">Device Connection</p>
              <h3 className="mt-1 text-2xl font-bold text-gray-900">
                {device.connected ? 'Connected' : 'Not Connected'}
              </h3>
            </div>
            <StatusBadge status={device.status} />
          </div>

          <div className="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="rounded-xl bg-gray-50 p-4">
              <p className="text-xs text-gray-400">WhatsApp Number</p>
              <p className="mt-1 font-semibold text-gray-900">
                {device.phone || '—'}
              </p>
            </div>
            <div className="rounded-xl bg-gray-50 p-4">
              <p className="text-xs text-gray-400">Account Name</p>
              <p className="mt-1 font-semibold text-gray-900">
                {device.push_name || '—'}
              </p>
            </div>
          </div>

          {!device.connected && (
            <button
              onClick={() => navigate('/device')}
              className="mt-6 inline-flex items-center gap-2 rounded-xl bg-brand-500 px-4 py-2.5 text-sm font-semibold text-white hover:bg-brand-600"
            >
              <span className="icon-[tabler--qrcode] size-4" />
              Connect Device
            </button>
          )}
          {isLoading && <p className="mt-4 text-sm text-gray-400">Loading…</p>}
        </Card>

        {/* Quick actions (side) */}
        <Card className="p-5">
          <h3 className="font-semibold text-gray-900">Quick Actions</h3>
          <div className="mt-4 space-y-2">
            {[
              { to: '/chats', icon: 'icon-[tabler--message-circle]', label: 'Open Inbox' },
              { to: '/broadcast', icon: 'icon-[tabler--send]', label: 'New Broadcast' },
              { to: '/device', icon: 'icon-[tabler--device-mobile]', label: 'Device Settings' },
              { to: '/docs', icon: 'icon-[tabler--code]', label: 'API Reference' },
            ].map((a) => (
              <button
                key={a.to}
                onClick={() => navigate(a.to)}
                className="flex w-full items-center gap-3 rounded-xl border border-gray-100 px-3 py-2.5 text-left text-sm font-medium text-gray-700 hover:border-brand-200 hover:bg-brand-50"
              >
                <span className={`${a.icon} size-5 text-brand-600`} />
                {a.label}
                <span className="icon-[tabler--chevron-right] ml-auto size-4 text-gray-300" />
              </button>
            ))}
          </div>
        </Card>
      </div>
    </div>
  )
}
