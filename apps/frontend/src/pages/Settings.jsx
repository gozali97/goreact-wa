import { useQuery } from '@tanstack/react-query'
import { deviceApi } from '../lib/api.js'
import Card from '../components/Card.jsx'
import StatusBadge from '../components/StatusBadge.jsx'

function Row({ label, value }) {
  return (
    <div className="flex justify-between gap-4 border-b border-gray-100 py-2.5 text-sm last:border-0">
      <span className="text-gray-500">{label}</span>
      <span className="truncate text-right font-medium text-gray-900">{value ?? '—'}</span>
    </div>
  )
}

export default function Settings() {
  const { data } = useQuery({
    queryKey: ['device-status'],
    queryFn: deviceApi.status,
  })

  return (
    <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
      <Card className="p-5">
        <div className="flex items-center gap-3">
          <span className="flex h-10 w-10 items-center justify-center rounded-xl bg-brand-50">
            <span className="icon-[tabler--device-mobile] size-5 text-brand-600" />
          </span>
          <h3 className="text-lg font-semibold text-gray-900">Device Information</h3>
        </div>
        <div className="mt-4 flex items-center gap-3 pb-1">
          <span className="text-sm text-gray-500">Status</span>
          <StatusBadge status={data?.status} />
        </div>
        <Row label="Phone Number" value={data?.phone} />
        <Row label="Account Name" value={data?.push_name} />
        <Row label="JID" value={data?.jid} />
      </Card>

      <Card className="p-5">
        <div className="flex items-center gap-3">
          <span className="flex h-10 w-10 items-center justify-center rounded-xl bg-gray-100">
            <span className="icon-[tabler--adjustments] size-5 text-gray-600" />
          </span>
          <h3 className="text-lg font-semibold text-gray-900">App Configuration</h3>
        </div>
        <div className="mt-4">
          <Row label="Authentication" value="HTTP Basic Auth" />
          <Row label="Storage" value="Local filesystem (/storage)" />
          <Row label="Database" value="PostgreSQL" />
        </div>
        <p className="mt-3 text-xs text-gray-400">
          Configuration is managed via environment variables. See .env.example.
        </p>
      </Card>
    </div>
  )
}
