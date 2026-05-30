import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { deviceApi } from '../lib/api.js'
import Card from './Card.jsx'

// Shows the integration API key with generate / regenerate / copy / reveal.
export default function ApiKeyCard() {
  const queryClient = useQueryClient()
  const [revealed, setRevealed] = useState(false)
  const [copied, setCopied] = useState(false)
  const [confirmRegen, setConfirmRegen] = useState(false)

  const { data, isLoading } = useQuery({
    queryKey: ['api-key'],
    queryFn: deviceApi.getApiKey,
  })
  const key = data?.api_key || ''

  const genMut = useMutation({
    mutationFn: deviceApi.generateApiKey,
    onSuccess: (res) => {
      queryClient.setQueryData(['api-key'], res)
      setRevealed(true)
      setConfirmRegen(false)
    },
  })

  const masked = key ? key.slice(0, 6) + '•'.repeat(Math.max(0, key.length - 10)) + key.slice(-4) : ''

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(key)
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } catch {
      /* ignore */
    }
  }

  return (
    <Card className="p-5">
      <div className="flex items-center gap-3">
        <span className="flex h-10 w-10 items-center justify-center rounded-xl bg-brand-50">
          <span className="icon-[tabler--key] size-5 text-brand-600" />
        </span>
        <div>
          <h3 className="font-semibold text-gray-900">Integration API Key</h3>
          <p className="text-xs text-gray-400">Use as the X-Api-Key header for external apps</p>
        </div>
      </div>

      <div className="mt-4">
        {isLoading ? (
          <p className="text-sm text-gray-400">Loading…</p>
        ) : key ? (
          <>
            <div className="flex items-center gap-2 rounded-xl border border-gray-200 bg-gray-50 p-2">
              <code className="flex-1 truncate px-2 font-mono text-sm text-gray-700">
                {revealed ? key : masked}
              </code>
              <button
                onClick={() => setRevealed((v) => !v)}
                className="flex h-8 w-8 items-center justify-center rounded-lg text-gray-400 hover:bg-gray-200"
                title={revealed ? 'Hide' : 'Reveal'}
              >
                <span className={`size-4 ${revealed ? 'icon-[tabler--eye-off]' : 'icon-[tabler--eye]'}`} />
              </button>
              <button
                onClick={copy}
                className="inline-flex items-center gap-1.5 rounded-lg bg-brand-500 px-3 py-1.5 text-sm font-medium text-white hover:bg-brand-600"
              >
                <span className={`size-4 ${copied ? 'icon-[tabler--check]' : 'icon-[tabler--copy]'}`} />
                {copied ? 'Copied' : 'Copy'}
              </button>
            </div>

            <div className="mt-3 flex items-center gap-3">
              {!confirmRegen ? (
                <button
                  onClick={() => setConfirmRegen(true)}
                  className="inline-flex items-center gap-1.5 text-sm font-medium text-gray-500 hover:text-gray-800"
                >
                  <span className="icon-[tabler--refresh] size-4" />
                  Regenerate
                </button>
              ) : (
                <div className="flex items-center gap-2 text-sm">
                  <span className="text-amber-600">Old key stops working immediately.</span>
                  <button
                    onClick={() => genMut.mutate()}
                    disabled={genMut.isPending}
                    className="rounded-lg bg-red-500 px-3 py-1 font-medium text-white hover:bg-red-600 disabled:opacity-50"
                  >
                    {genMut.isPending ? '…' : 'Confirm'}
                  </button>
                  <button onClick={() => setConfirmRegen(false)} className="text-gray-500 hover:text-gray-800">
                    Cancel
                  </button>
                </div>
              )}
            </div>
          </>
        ) : (
          <div className="rounded-xl bg-gray-50 p-5 text-center">
            <p className="mb-3 text-sm text-gray-500">No API key yet. Generate one to let external apps call the messaging API.</p>
            <button
              onClick={() => genMut.mutate()}
              disabled={genMut.isPending}
              className="inline-flex items-center gap-2 rounded-xl bg-brand-500 px-4 py-2.5 text-sm font-semibold text-white hover:bg-brand-600 disabled:opacity-50"
            >
              <span className="icon-[tabler--key] size-4" />
              {genMut.isPending ? 'Generating…' : 'Generate API Key'}
            </button>
          </div>
        )}
      </div>
    </Card>
  )
}
