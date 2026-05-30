import { useState } from 'react'
import { runRequest } from '../lib/api.js'
import CodeBlock from './CodeBlock.jsx'

// Renders the "Try it" form for an endpoint's `try` descriptor and executes the
// live request through the authed client.
export default function TryItPanel({ tryDef }) {
  const [values, setValues] = useState(() => {
    const init = {}
    for (const f of tryDef.fields) {
      if (f.type === 'boolean') init[f.name] = f.default ?? false
      else init[f.name] = f.default ?? ''
    }
    return init
  })
  const [files, setFiles] = useState({})
  const [result, setResult] = useState(null)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const setVal = (name, v) => setValues((s) => ({ ...s, [name]: v }))

  const send = async () => {
    setError('')
    setResult(null)

    // Resolve path params.
    let path = tryDef.path
    const query = {}
    let body
    let formData

    try {
      for (const f of tryDef.fields) {
        const raw = values[f.name]
        if (f.in === 'path') {
          if (f.required && (raw === '' || raw == null)) throw new Error(`${f.name} is required`)
          path = path.replace(`{${f.name}}`, encodeURIComponent(raw))
        } else if (f.in === 'query') {
          if (raw !== '' && raw != null) query[f.name] = raw
        } else if (f.in === 'form') {
          if (!formData) formData = new FormData()
          if (f.type === 'file') {
            if (files[f.name]) formData.append(f.name, files[f.name])
            else if (f.required) throw new Error(`${f.name} (file) is required`)
          } else if (raw !== '' && raw != null) {
            formData.append(f.name, raw)
          }
        } else if (f.in === 'body') {
          if (!body) body = {}
          if (f.type === 'json') {
            if (raw !== '' && raw != null) {
              try {
                body[f.name] = JSON.parse(raw)
              } catch {
                throw new Error(`${f.name} must be valid JSON`)
              }
            }
          } else if (f.type === 'boolean') {
            body[f.name] = !!raw
          } else if (f.type === 'number') {
            if (raw !== '' && raw != null) body[f.name] = Number(raw)
          } else if (raw !== '' && raw != null) {
            body[f.name] = raw
          }
        }
      }
    } catch (e) {
      setError(e.message)
      return
    }

    setLoading(true)
    try {
      const res = await runRequest({
        method: tryDef.method,
        path,
        query: Object.keys(query).length ? query : undefined,
        body,
        formData,
        contentType: tryDef.contentType,
      })
      setResult(res)
    } catch (e) {
      setError(e?.message || 'Request failed')
    } finally {
      setLoading(false)
    }
  }

  const statusColor = (s) =>
    s >= 200 && s < 300
      ? 'bg-brand-100 text-brand-700'
      : s >= 400 && s < 500
        ? 'bg-amber-100 text-amber-700'
        : 'bg-red-100 text-red-700'

  return (
    <div className="rounded-xl border border-gray-200 p-4">
      <div className="mb-3 flex items-center gap-2">
        <span className="icon-[tabler--player-play] size-4 text-brand-600" />
        <h4 className="text-sm font-semibold text-gray-900">Try it</h4>
      </div>

      {tryDef.fields.length > 0 && (
        <div className="space-y-3">
          {tryDef.fields.map((f) => (
            <div key={f.name}>
              <label className="mb-1 flex items-center gap-2 text-xs font-medium text-gray-600">
                {f.name}
                <span className="rounded bg-gray-100 px-1.5 py-0.5 text-[10px] text-gray-400">{f.in}</span>
                {f.required && <span className="text-red-500">*</span>}
              </label>
              {f.type === 'textarea' || f.type === 'json' ? (
                <textarea
                  rows={f.type === 'json' ? 3 : 2}
                  value={values[f.name]}
                  onChange={(e) => setVal(f.name, e.target.value)}
                  placeholder={f.placeholder || f.help}
                  className="w-full rounded-lg border border-gray-200 px-3 py-2 font-mono text-xs focus:border-brand-500 focus:outline-none"
                />
              ) : f.type === 'file' ? (
                <input
                  type="file"
                  onChange={(e) => setFiles((s) => ({ ...s, [f.name]: e.target.files?.[0] }))}
                  className="w-full text-xs text-gray-600 file:mr-3 file:rounded-lg file:border-0 file:bg-brand-50 file:px-3 file:py-1.5 file:text-xs file:font-medium file:text-brand-700"
                />
              ) : f.type === 'boolean' ? (
                <label className="flex items-center gap-2 text-sm text-gray-700">
                  <input
                    type="checkbox"
                    checked={!!values[f.name]}
                    onChange={(e) => setVal(f.name, e.target.checked)}
                    className="h-4 w-4 rounded border-gray-300 text-brand-500"
                  />
                  {String(!!values[f.name])}
                </label>
              ) : (
                <input
                  type={f.type === 'number' ? 'number' : 'text'}
                  value={values[f.name]}
                  onChange={(e) => setVal(f.name, e.target.value)}
                  placeholder={f.placeholder || f.help}
                  className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none"
                />
              )}
              {f.help && <p className="mt-1 text-[11px] text-gray-400">{f.help}</p>}
            </div>
          ))}
        </div>
      )}

      <button
        onClick={send}
        disabled={loading}
        className="mt-4 inline-flex items-center gap-2 rounded-lg bg-brand-500 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-600 disabled:opacity-50"
      >
        {loading ? (
          <span className="icon-[svg-spinners--ring-resize] size-4" />
        ) : (
          <span className="icon-[tabler--send] size-4" />
        )}
        Send request
      </button>

      {error && (
        <div className="mt-3 flex items-center gap-2 rounded-lg bg-red-50 px-3 py-2 text-sm text-red-600">
          <span className="icon-[tabler--alert-circle] size-4" />
          {error}
        </div>
      )}

      {result && (
        <div className="mt-4">
          <div className="mb-2 flex items-center gap-3 text-sm">
            <span className={`rounded-full px-2.5 py-0.5 text-xs font-semibold ${statusColor(result.status)}`}>
              {result.status} {result.statusText}
            </span>
            <span className="text-gray-400">{result.durationMs} ms</span>
          </div>
          <CodeBlock code={result.data} language="response" />
        </div>
      )}
    </div>
  )
}
