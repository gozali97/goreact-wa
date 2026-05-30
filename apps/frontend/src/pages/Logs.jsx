import { useEffect, useState } from 'react'
import { useQuery, keepPreviousData } from '@tanstack/react-query'
import { logApi } from '../lib/api.js'
import Card from '../components/Card.jsx'

const STATUS_STYLE = {
  success: 'bg-brand-100 text-brand-700',
  failed: 'bg-red-100 text-red-700',
  issue: 'bg-amber-100 text-amber-700',
  info: 'bg-blue-100 text-blue-700',
}

const STATUS_FILTERS = [
  { value: '', label: 'All' },
  { value: 'success', label: 'Success' },
  { value: 'failed', label: 'Failed' },
  { value: 'issue', label: 'Issue' },
  { value: 'info', label: 'Info' },
]

function StatusPill({ status }) {
  return (
    <span className={`rounded-full px-2 py-0.5 text-xs font-medium capitalize ${STATUS_STYLE[status] || STATUS_STYLE.info}`}>
      {status}
    </span>
  )
}

function fmt(ts) {
  if (!ts) return ''
  return new Date(ts).toLocaleString([], {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

// Detail modal for a single entry, with copy.
function DetailModal({ entry, onClose }) {
  const [copied, setCopied] = useState(false)
  if (!entry) return null

  const json = JSON.stringify(entry, null, 2)
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(json)
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } catch {
      /* ignore */
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" onClick={onClose}>
      <Card className="flex max-h-[85vh] w-full max-w-2xl flex-col" >
        <div
          className="flex items-center justify-between border-b border-gray-100 p-4"
          onClick={(e) => e.stopPropagation()}
        >
          <div className="flex items-center gap-2">
            <StatusPill status={entry.status} />
            <h3 className="font-semibold text-gray-900">{entry.source}</h3>
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={copy}
              className="inline-flex items-center gap-1.5 rounded-lg border border-gray-200 px-3 py-1.5 text-sm font-medium text-gray-600 hover:bg-gray-50"
            >
              <span className={`size-4 ${copied ? 'icon-[tabler--check] text-brand-600' : 'icon-[tabler--copy]'}`} />
              {copied ? 'Copied' : 'Copy'}
            </button>
            <button
              onClick={onClose}
              className="flex h-8 w-8 items-center justify-center rounded-lg text-gray-400 hover:bg-gray-100"
              aria-label="Close"
            >
              <span className="icon-[tabler--x] size-5" />
            </button>
          </div>
        </div>
        <div className="overflow-y-auto p-4" onClick={(e) => e.stopPropagation()}>
          <div className="mb-3 grid grid-cols-2 gap-3 text-sm">
            <div>
              <p className="text-xs text-gray-400">Time</p>
              <p className="font-medium">{new Date(entry.time).toLocaleString()}</p>
            </div>
            <div>
              <p className="text-xs text-gray-400">Level</p>
              <p className="font-medium capitalize">{entry.level}</p>
            </div>
            <div className="col-span-2">
              <p className="text-xs text-gray-400">Message</p>
              <p className="font-medium">{entry.message}</p>
            </div>
          </div>
          <p className="mb-1 text-xs font-medium text-gray-400">RAW</p>
          <pre className="overflow-x-auto rounded-xl bg-gray-900 p-3 text-xs text-gray-100">{json}</pre>
        </div>
      </Card>
    </div>
  )
}

export default function Logs() {
  const [day, setDay] = useState('')
  const [search, setSearch] = useState('')
  const [debounced, setDebounced] = useState('')
  const [status, setStatus] = useState('')
  const [page, setPage] = useState(1)
  const [perPage, setPerPage] = useState(50)
  const [detail, setDetail] = useState(null)

  // Debounce the search box.
  useEffect(() => {
    const t = setTimeout(() => setDebounced(search), 350)
    return () => clearTimeout(t)
  }, [search])

  // Reset to page 1 when filters change.
  useEffect(() => {
    setPage(1)
  }, [debounced, status, day, perPage])

  const { data: daysData } = useQuery({
    queryKey: ['log-days'],
    queryFn: logApi.days,
    refetchInterval: 30000,
  })
  const days = daysData?.days || []

  // Default to the newest day once loaded.
  useEffect(() => {
    if (!day && days.length > 0) setDay(days[0])
  }, [days, day])

  const { data, isFetching } = useQuery({
    queryKey: ['logs', day, debounced, status, page, perPage],
    queryFn: () =>
      logApi.entries({
        day,
        search: debounced,
        status,
        page,
        per_page: perPage,
      }),
    placeholderData: keepPreviousData,
    refetchInterval: 10000,
  })

  const result = data?.result
  const entries = result?.entries || []
  const pages = result?.pages || 1
  const total = result?.total || 0

  return (
    <div className="space-y-4">
      {/* Toolbar */}
      <Card className="p-4">
        <div className="flex flex-wrap items-center gap-3">
          {/* Day */}
          <div className="flex items-center gap-2">
            <span className="icon-[tabler--calendar] size-5 text-gray-400" />
            <select
              value={day}
              onChange={(e) => setDay(e.target.value)}
              className="rounded-xl border border-gray-200 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none"
            >
              {days.length === 0 && <option value="">No logs yet</option>}
              {days.map((d) => (
                <option key={d} value={d}>
                  {d}
                </option>
              ))}
            </select>
          </div>

          {/* Search */}
          <div className="relative min-w-[200px] flex-1">
            <span className="icon-[tabler--search] absolute left-3 top-1/2 size-4 -translate-y-1/2 text-gray-400" />
            <input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search message, source, fields…"
              className="w-full rounded-xl border border-gray-200 py-2 pl-9 pr-3 text-sm focus:border-brand-500 focus:outline-none"
            />
          </div>

          {/* Status filter */}
          <div className="flex items-center gap-1 rounded-xl bg-gray-100 p-1">
            {STATUS_FILTERS.map((f) => (
              <button
                key={f.value}
                onClick={() => setStatus(f.value)}
                className={`rounded-lg px-3 py-1.5 text-xs font-medium transition ${
                  status === f.value ? 'bg-white text-gray-900 shadow-sm' : 'text-gray-500 hover:text-gray-700'
                }`}
              >
                {f.label}
              </button>
            ))}
          </div>

          {/* Download */}
          <a
            href={logApi.downloadUrl(day)}
            className="inline-flex items-center gap-1.5 rounded-xl bg-brand-500 px-3 py-2 text-sm font-semibold text-white hover:bg-brand-600"
          >
            <span className="icon-[tabler--download] size-4" />
            Download
          </a>
        </div>
      </Card>

      {/* Table */}
      <Card className="overflow-hidden">
        <div className="flex items-center justify-between border-b border-gray-100 px-4 py-3">
          <p className="text-sm text-gray-500">
            {total} {total === 1 ? 'entry' : 'entries'}
            {isFetching && <span className="ml-2 text-gray-300">updating…</span>}
          </p>
          <div className="flex items-center gap-2 text-sm">
            <span className="text-gray-400">Per page</span>
            <select
              value={perPage}
              onChange={(e) => setPerPage(Number(e.target.value))}
              className="rounded-lg border border-gray-200 px-2 py-1 text-sm focus:outline-none"
            >
              {[25, 50, 100, 200].map((n) => (
                <option key={n} value={n}>
                  {n}
                </option>
              ))}
            </select>
          </div>
        </div>

        <div className="divide-y divide-gray-50">
          {entries.length === 0 && (
            <div className="flex flex-col items-center gap-2 p-10 text-center">
              <span className="icon-[tabler--file-search] size-8 text-gray-300" />
              <p className="text-sm text-gray-400">No log entries match your filters.</p>
            </div>
          )}
          {entries.map((e, i) => (
            <button
              key={`${e.time}-${i}`}
              onClick={() => setDetail(e)}
              className="flex w-full items-center gap-3 px-4 py-2.5 text-left hover:bg-gray-50"
            >
              <span className="w-20 flex-none font-mono text-xs text-gray-400">{fmt(e.time)}</span>
              <span className="w-20 flex-none">
                <StatusPill status={e.status} />
              </span>
              <span className="w-48 flex-none truncate text-sm font-medium text-gray-700">{e.source}</span>
              <span className="min-w-0 flex-1 truncate text-sm text-gray-600">{e.message}</span>
              <span className="icon-[tabler--chevron-right] size-4 flex-none text-gray-300" />
            </button>
          ))}
        </div>

        {/* Pagination */}
        {pages > 1 && (
          <div className="flex items-center justify-between border-t border-gray-100 px-4 py-3">
            <button
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              disabled={page <= 1}
              className="inline-flex items-center gap-1 rounded-lg border border-gray-200 px-3 py-1.5 text-sm font-medium text-gray-600 hover:bg-gray-50 disabled:opacity-40"
            >
              <span className="icon-[tabler--chevron-left] size-4" />
              Prev
            </button>
            <span className="text-sm text-gray-500">
              Page {page} of {pages}
            </span>
            <button
              onClick={() => setPage((p) => Math.min(pages, p + 1))}
              disabled={page >= pages}
              className="inline-flex items-center gap-1 rounded-lg border border-gray-200 px-3 py-1.5 text-sm font-medium text-gray-600 hover:bg-gray-50 disabled:opacity-40"
            >
              Next
              <span className="icon-[tabler--chevron-right] size-4" />
            </button>
          </div>
        )}
      </Card>

      <DetailModal entry={detail} onClose={() => setDetail(null)} />
    </div>
  )
}
