// Generates curl and JavaScript (fetch) code samples for an endpoint spec.

const ORIGIN = typeof window !== 'undefined' ? window.location.origin : 'http://localhost:8080'

function fullUrl(path) {
  return ORIGIN + path.replace(/\{(\w+)\}/g, ':$1')
}

export function curlSample(ep) {
  const url = fullUrl(ep.path)
  const lines = [`curl -X ${ep.method} '${url}' \\`]
  lines.push(`  -H 'X-Api-Key: YOUR_API_KEY' \\`)

  if (ep.contentType === 'multipart/form-data' && ep.bodySchema) {
    const parts = ep.bodySchema.map((f) =>
      f.type === 'file' ? `  -F '${f.name}=@/path/to/file'` : `  -F '${f.name}=value'`
    )
    return [`curl -X ${ep.method} '${url}' \\`, `  -H 'X-Api-Key: YOUR_API_KEY' \\`, parts.join(' \\\n')].join('\n')
  }

  if (ep.body) {
    lines.push(`  -H 'Content-Type: application/json' \\`)
    lines.push(`  -d '${JSON.stringify(ep.body)}'`)
  } else {
    // Trim trailing backslash on the last header line.
    lines[lines.length - 1] = lines[lines.length - 1].replace(/ \\$/, '')
  }
  return lines.join('\n')
}

export function jsSample(ep) {
  const url = fullUrl(ep.path)
  if (ep.contentType === 'multipart/form-data' && ep.bodySchema) {
    const appends = ep.bodySchema
      .map((f) =>
        f.type === 'file'
          ? `form.append('${f.name}', fileInput.files[0])`
          : `form.append('${f.name}', 'value')`
      )
      .join('\n')
    return `const form = new FormData()
${appends}

const res = await fetch('${url}', {
  method: '${ep.method}',
  headers: { 'X-Api-Key': 'YOUR_API_KEY' },
  body: form,
})
const data = await res.json()`
  }

  const opts = [`  method: '${ep.method}'`, `  headers: {\n    'X-Api-Key': 'YOUR_API_KEY'${ep.body ? ",\n    'Content-Type': 'application/json'" : ''}\n  }`]
  if (ep.body) opts.push(`  body: JSON.stringify(${JSON.stringify(ep.body, null, 2).replace(/\n/g, '\n  ')})`)

  return `const res = await fetch('${url}', {
${opts.join(',\n')}
})
const data = await res.json()`
}
