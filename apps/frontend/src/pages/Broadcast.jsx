import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { broadcastApi } from '../lib/api.js'
import Card from '../components/Card.jsx'

const STATUS_STYLE = {
  pending: 'bg-gray-100 text-gray-600',
  sending: 'bg-amber-100 text-amber-700',
  success: 'bg-brand-100 text-brand-700',
  failed: 'bg-red-100 text-red-700',
}

function StatusPill({ status }) {
  return (
    <span className={`rounded-full px-2.5 py-0.5 text-xs font-medium capitalize ${STATUS_STYLE[status] || STATUS_STYLE.pending}`}>
      {status}
    </span>
  )
}

export default function Broadcast() {
  const queryClient = useQueryClient()
  const [name, setName] = useState('')
  const [message, setMessage] = useState('')
  const [phonesText, setPhonesText] = useState('')
  const [feedback, setFeedback] = useState('')

  const { data } = useQuery({
    queryKey: ['broadcasts'],
    queryFn: broadcastApi.list,
    refetchInterval: 10000,
  })
  const broadcasts = data?.broadcasts || []

  const createMut = useMutation({
    mutationFn: broadcastApi.create,
    onSuccess: (res) => {
      setFeedback(`Broadcast queued to ${res.total_target} recipients.`)
      setName('')
      setMessage('')
      setPhonesText('')
      queryClient.invalidateQueries({ queryKey: ['broadcasts'] })
    },
    onError: (e) => setFeedback(e?.response?.data?.error || e.message),
  })

  const phoneCount = phonesText
    .split(/[\n,;]+/)
    .map((p) => p.trim())
    .filter(Boolean).length

  const handleSubmit = (e) => {
    e.preventDefault()
    const phones = phonesText
      .split(/[\n,;]+/)
      .map((p) => p.trim())
      .filter(Boolean)
    if (!message.trim()) {
      setFeedback('Message is required.')
      return
    }
    // phones optional: empty → backend sends to all saved contacts.
    createMut.mutate({ name: name.trim(), message: message.trim(), phones })
  }

  return (
    <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
      {/* Create form */}
      <Card className="p-5">
        <h3 className="text-lg font-semibold text-gray-900">Create Broadcast</h3>
        <form onSubmit={handleSubmit} className="mt-4 space-y-4">
          <div>
            <label className="mb-1 block text-sm font-medium text-gray-700">Campaign name</label>
            <input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Promo Hari Ini"
              className="w-full rounded-xl border border-gray-200 px-3 py-2.5 text-sm focus:border-brand-500 focus:outline-none"
            />
          </div>

          <div>
            <label className="mb-1 block text-sm font-medium text-gray-700">Message</label>
            <textarea
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              rows={4}
              placeholder="Your message…"
              className="w-full rounded-xl border border-gray-200 px-3 py-2.5 text-sm focus:border-brand-500 focus:outline-none"
            />
          </div>

          <div>
            <div className="mb-1 flex items-center justify-between">
              <label className="block text-sm font-medium text-gray-700">
                Phone numbers <span className="font-normal text-gray-400">(optional)</span>
              </label>
              {phoneCount > 0 && (
                <span className="text-xs text-gray-400">{phoneCount} recipients</span>
              )}
            </div>
            <textarea
              value={phonesText}
              onChange={(e) => setPhonesText(e.target.value)}
              rows={5}
              placeholder={'628111111111\n628222222222'}
              className="w-full rounded-xl border border-gray-200 px-3 py-2.5 font-mono text-sm focus:border-brand-500 focus:outline-none"
            />
            <p className="mt-1 text-xs text-gray-400">
              One per line, or comma/semicolon separated. Leave empty to send to
              all saved contacts. Numbers not on WhatsApp are skipped automatically.
            </p>
          </div>

          {feedback && (
            <div className="rounded-xl bg-gray-50 px-3 py-2.5 text-sm text-gray-600">{feedback}</div>
          )}

          <button
            type="submit"
            disabled={createMut.isPending}
            className="inline-flex items-center gap-2 rounded-xl bg-brand-500 px-4 py-2.5 text-sm font-semibold text-white hover:bg-brand-600 disabled:opacity-50"
          >
            <span className="icon-[tabler--send] size-4" />
            {createMut.isPending ? 'Queuing…' : 'Send Broadcast'}
          </button>
        </form>
      </Card>

      {/* History */}
      <Card className="p-5">
        <h3 className="text-lg font-semibold text-gray-900">History</h3>
        <div className="mt-4 space-y-3">
          {broadcasts.length === 0 && (
            <div className="flex flex-col items-center gap-2 p-8 text-center">
              <span className="icon-[tabler--inbox] size-8 text-gray-300" />
              <p className="text-sm text-gray-400">No broadcasts yet.</p>
            </div>
          )}
          {broadcasts.map((b) => (
            <div key={b.id} className="rounded-xl border border-gray-100 p-4">
              <div className="flex items-center justify-between gap-2">
                <span className="truncate font-medium text-gray-900">{b.name}</span>
                <StatusPill status={b.status} />
              </div>
              <p className="mt-1 truncate text-sm text-gray-500">{b.message}</p>
              <div className="mt-3 flex gap-4 text-xs">
                <span className="flex items-center gap-1 text-gray-500">
                  <span className="icon-[tabler--target] size-4" /> {b.total_target}
                </span>
                <span className="flex items-center gap-1 text-brand-600">
                  <span className="icon-[tabler--circle-check] size-4" /> {b.total_success}
                </span>
                <span className="flex items-center gap-1 text-red-500">
                  <span className="icon-[tabler--circle-x] size-4" /> {b.total_failed}
                </span>
              </div>
            </div>
          ))}
        </div>
      </Card>
    </div>
  )
}
