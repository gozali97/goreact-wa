const STYLES = {
  connected: 'bg-brand-100 text-brand-700',
  connecting: 'bg-amber-100 text-amber-700',
  disconnected: 'bg-gray-200 text-gray-600',
  logged_out: 'bg-red-100 text-red-700',
  unknown: 'bg-gray-100 text-gray-500',
}

const LABELS = {
  connected: 'Connected',
  connecting: 'Connecting',
  disconnected: 'Disconnected',
  logged_out: 'Logged Out',
  unknown: 'Unknown',
}

const DOT = {
  connected: 'bg-brand-500',
  connecting: 'bg-amber-500 animate-pulse',
  disconnected: 'bg-gray-400',
  logged_out: 'bg-red-500',
  unknown: 'bg-gray-400',
}

export default function StatusBadge({ status }) {
  const key = status || 'unknown'
  return (
    <span
      className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-medium ${
        STYLES[key] || STYLES.unknown
      }`}
    >
      <span className={`h-2 w-2 rounded-full ${DOT[key] || DOT.unknown}`} />
      {LABELS[key] || LABELS.unknown}
    </span>
  )
}
