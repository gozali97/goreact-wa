import { useState } from 'react'
import Card from '../components/Card.jsx'
import CodeBlock from '../components/CodeBlock.jsx'
import TryItPanel from '../components/TryItPanel.jsx'
import { API_SPEC } from '../lib/apiSpec.js'
import { curlSample, jsSample } from '../lib/samples.js'

const METHOD_STYLE = {
  GET: 'bg-blue-100 text-blue-700',
  POST: 'bg-brand-100 text-brand-700',
  PUT: 'bg-amber-100 text-amber-700',
  DELETE: 'bg-red-100 text-red-700',
}

function MethodBadge({ method }) {
  return (
    <span className={`rounded-md px-2 py-0.5 text-xs font-bold ${METHOD_STYLE[method] || 'bg-gray-100 text-gray-600'}`}>
      {method}
    </span>
  )
}

function ParamTable({ title, rows }) {
  if (!rows || rows.length === 0) return null
  return (
    <div className="mt-4">
      <p className="mb-1.5 text-xs font-semibold uppercase tracking-wide text-gray-400">{title}</p>
      <div className="overflow-hidden rounded-xl border border-gray-100">
        <table className="w-full text-left text-sm">
          <thead className="bg-gray-50 text-xs text-gray-500">
            <tr>
              <th className="px-3 py-2 font-medium">Name</th>
              <th className="px-3 py-2 font-medium">Type</th>
              <th className="px-3 py-2 font-medium">Required</th>
              <th className="px-3 py-2 font-medium">Description</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-50">
            {rows.map((r) => (
              <tr key={r.name}>
                <td className="px-3 py-2 font-mono text-xs text-gray-800">{r.name}</td>
                <td className="px-3 py-2 text-xs text-gray-500">{r.type}</td>
                <td className="px-3 py-2 text-xs">
                  {r.required ? (
                    <span className="text-red-500">required</span>
                  ) : (
                    <span className="text-gray-400">optional</span>
                  )}
                </td>
                <td className="px-3 py-2 text-xs text-gray-600">{r.desc}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

function Samples({ ep }) {
  const [tab, setTab] = useState('curl')
  const tabs = [
    { id: 'curl', label: 'cURL' },
    { id: 'js', label: 'JavaScript' },
  ]
  const code = tab === 'curl' ? curlSample(ep) : jsSample(ep)
  return (
    <div className="mt-4">
      <div className="mb-1.5 flex items-center justify-between">
        <p className="text-xs font-semibold uppercase tracking-wide text-gray-400">Request sample</p>
        <div className="flex gap-1 rounded-lg bg-gray-100 p-0.5">
          {tabs.map((t) => (
            <button
              key={t.id}
              onClick={() => setTab(t.id)}
              className={`rounded-md px-2.5 py-1 text-xs font-medium transition ${
                tab === t.id ? 'bg-white text-gray-900 shadow-sm' : 'text-gray-500'
              }`}
            >
              {t.label}
            </button>
          ))}
        </div>
      </div>
      <CodeBlock code={code} language={tab === 'curl' ? 'bash' : 'javascript'} />
    </div>
  )
}

function EndpointCard({ ep }) {
  const [open, setOpen] = useState(false)
  return (
    <Card id={ep.id} className="scroll-mt-24 overflow-hidden">
      <button
        onClick={() => setOpen((v) => !v)}
        className="flex w-full items-center gap-3 px-5 py-4 text-left"
      >
        <MethodBadge method={ep.method} />
        <code className="text-sm text-gray-700">{ep.path}</code>
        <span className="ml-1 hidden text-sm font-medium text-gray-900 sm:inline">— {ep.title}</span>
        <span
          className={`icon-[tabler--chevron-down] ml-auto size-5 text-gray-400 transition ${open ? 'rotate-180' : ''}`}
        />
      </button>

      {open && (
        <div className="border-t border-gray-100 px-5 py-4">
          <p className="text-sm text-gray-600">{ep.description}</p>
          {ep.auth && (
            <p className="mt-2 inline-flex items-center gap-1.5 rounded-lg bg-gray-50 px-2.5 py-1 text-xs text-gray-500">
              <span className="icon-[tabler--lock] size-3.5" />
              Auth: {ep.auth}
            </p>
          )}
          {ep.contentType && (
            <p className="mt-2 ml-2 inline-flex items-center gap-1.5 rounded-lg bg-gray-50 px-2.5 py-1 text-xs text-gray-500">
              <span className="icon-[tabler--file-upload] size-3.5" />
              {ep.contentType}
            </p>
          )}

          <ParamTable title="Path & query parameters" rows={ep.params} />
          <ParamTable title="Request body" rows={ep.bodySchema} />

          {ep.body && (
            <div className="mt-4">
              <p className="mb-1.5 text-xs font-semibold uppercase tracking-wide text-gray-400">Body example</p>
              <CodeBlock code={ep.body} language="json" />
            </div>
          )}

          {!ep.noTry && ep.method && ep.path.startsWith('/api') && <Samples ep={ep} />}

          {ep.responses?.map((r, i) => (
            <div key={i} className="mt-4">
              <p className="mb-1.5 flex items-center gap-2 text-xs font-semibold uppercase tracking-wide text-gray-400">
                <span
                  className={`rounded px-1.5 py-0.5 ${
                    r.status >= 200 && r.status < 300
                      ? 'bg-brand-100 text-brand-700'
                      : 'bg-amber-100 text-amber-700'
                  }`}
                >
                  {r.status}
                </span>
                {r.label || 'Response'}
              </p>
              <CodeBlock code={r.body} language="json" dark={false} />
            </div>
          ))}

          {ep.try && !ep.noTry && (
            <div className="mt-5">
              <TryItPanel tryDef={ep.try} />
            </div>
          )}
        </div>
      )}
    </Card>
  )
}

export default function ApiDocs() {
  const [activeModule, setActiveModule] = useState(API_SPEC[0].id)

  const scrollTo = (id) => {
    setActiveModule(id)
    document.getElementById('mod-' + id)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
  }

  return (
    <div className="flex gap-5">
      {/* Module nav */}
      <aside className="hidden w-56 flex-none lg:block">
        <div className="sticky top-0 space-y-1">
          <p className="px-3 pb-1 text-[11px] font-semibold uppercase tracking-wider text-gray-500">Modules</p>
          {API_SPEC.map((m) => (
            <button
              key={m.id}
              onClick={() => scrollTo(m.id)}
              className={`flex w-full items-center gap-2.5 rounded-xl px-3 py-2 text-left text-sm font-medium transition ${
                activeModule === m.id ? 'bg-white text-brand-700 shadow-sm' : 'text-gray-300 hover:bg-white/5 hover:text-white'
              }`}
            >
              <span className={`${m.icon} size-4`} />
              {m.name}
            </button>
          ))}
        </div>
      </aside>

      {/* Content */}
      <div className="min-w-0 flex-1 space-y-8">
        {/* Intro */}
        <Card className="p-5">
          <h2 className="text-lg font-semibold text-gray-900">WA Proxy API</h2>
          <p className="mt-1 text-sm text-gray-600">
            Base URL <code className="rounded bg-gray-100 px-1.5 py-0.5 text-xs">{window.location.origin}/api</code>.
            Authenticate with the <code className="rounded bg-gray-100 px-1.5 py-0.5 text-xs">X-Api-Key</code> header
            (set API_KEY in env) or dashboard Basic Auth. The interactive “Try it” panels run against your live server
            using your current session.
          </p>
        </Card>

        {API_SPEC.map((m) => (
          <section key={m.id} id={'mod-' + m.id} className="scroll-mt-4 space-y-3">
            <div className="flex items-center gap-3">
              <span className="flex h-9 w-9 items-center justify-center rounded-xl bg-white">
                <span className={`${m.icon} size-5 text-brand-600`} />
              </span>
              <div>
                <h3 className="text-lg font-semibold text-white">{m.name}</h3>
              </div>
            </div>
            <p className="text-sm text-gray-400">{m.description}</p>
            <div className="space-y-3">
              {m.endpoints.map((ep) => (
                <EndpointCard key={ep.id} ep={ep} />
              ))}
            </div>
          </section>
        ))}
      </div>
    </div>
  )
}
