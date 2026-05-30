import { useState } from 'react'

// A code block with a copy button. `dark` (default) renders on a dark surface.
export default function CodeBlock({ code, language = '', dark = true, className = '' }) {
  const [copied, setCopied] = useState(false)
  const text = typeof code === 'string' ? code : JSON.stringify(code, null, 2)

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(text)
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } catch {
      /* ignore */
    }
  }

  return (
    <div className={`group relative ${className}`}>
      <button
        onClick={copy}
        className={`absolute right-2 top-2 z-10 inline-flex items-center gap-1 rounded-lg px-2 py-1 text-xs font-medium opacity-0 transition group-hover:opacity-100 ${
          dark ? 'bg-white/10 text-white hover:bg-white/20' : 'bg-gray-200 text-gray-700 hover:bg-gray-300'
        }`}
        title="Copy"
      >
        <span className={`size-3.5 ${copied ? 'icon-[tabler--check]' : 'icon-[tabler--copy]'}`} />
        {copied ? 'Copied' : 'Copy'}
      </button>
      {language && (
        <span
          className={`absolute left-3 top-2 z-10 text-[10px] font-semibold uppercase tracking-wide ${
            dark ? 'text-gray-500' : 'text-gray-400'
          }`}
        >
          {language}
        </span>
      )}
      <pre
        className={`overflow-x-auto rounded-xl p-3 ${language ? 'pt-7' : ''} text-xs ${
          dark ? 'bg-gray-900 text-gray-100' : 'bg-gray-50 text-gray-700'
        }`}
      >
        {text}
      </pre>
    </div>
  )
}
