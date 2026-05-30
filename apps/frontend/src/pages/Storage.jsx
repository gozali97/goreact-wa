import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { storageApi, mediaUrl, downloadMedia } from '../lib/api.js'
import Card from '../components/Card.jsx'
import Lightbox from '../components/Lightbox.jsx'

function fmtBytes(n) {
  if (!n) return '0 B'
  const u = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(n) / Math.log(1024))
  return `${(n / Math.pow(1024, i)).toFixed(i === 0 ? 0 : 1)} ${u[i]}`
}

function fmtDate(ts) {
  if (!ts) return ''
  return new Date(ts * 1000).toLocaleString([], {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

const CAT_META = {
  images: { label: 'Images', icon: 'icon-[tabler--photo]', color: 'text-blue-600', bg: 'bg-blue-50' },
  docs: { label: 'Documents', icon: 'icon-[tabler--file-text]', color: 'text-amber-600', bg: 'bg-amber-50' },
  media: { label: 'Media (video/audio)', icon: 'icon-[tabler--movie]', color: 'text-purple-600', bg: 'bg-purple-50' },
}

function isImage(name) {
  return /\.(jpe?g|png|gif|webp|bmp)$/i.test(name)
}

function ConfirmButton({ label, danger, onConfirm, pending, icon }) {
  const [armed, setArmed] = useState(false)
  if (!armed) {
    return (
      <button
        onClick={() => setArmed(true)}
        className={`inline-flex items-center gap-1.5 rounded-lg border px-3 py-1.5 text-sm font-medium ${
          danger ? 'border-red-200 text-red-600 hover:bg-red-50' : 'border-gray-200 text-gray-600 hover:bg-gray-50'
        }`}
      >
        {icon && <span className={`${icon} size-4`} />}
        {label}
      </button>
    )
  }
  return (
    <div className="inline-flex items-center gap-2 text-sm">
      <span className="text-gray-500">Sure?</span>
      <button
        onClick={() => { onConfirm(); setArmed(false) }}
        disabled={pending}
        className="rounded-lg bg-red-500 px-3 py-1.5 font-medium text-white hover:bg-red-600 disabled:opacity-50"
      >
        Yes
      </button>
      <button onClick={() => setArmed(false)} className="text-gray-500 hover:text-gray-800">No</button>
    </div>
  )
}

export default function Storage() {
  const queryClient = useQueryClient()
  const [active, setActive] = useState('images')
  const [preview, setPreview] = useState(null)

  const { data: overview } = useQuery({
    queryKey: ['storage-overview'],
    queryFn: storageApi.overview,
    refetchInterval: 15000,
  })

  const { data: filesData } = useQuery({
    queryKey: ['storage-files', active],
    queryFn: () => storageApi.files(active),
    refetchInterval: 20000,
  })
  const files = filesData?.files || []

  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: ['storage-overview'] })
    queryClient.invalidateQueries({ queryKey: ['storage-files'] })
  }

  const delMut = useMutation({
    mutationFn: ({ category, name }) => storageApi.deleteFile(category, name),
    onSuccess: invalidate,
  })
  const clearCatMut = useMutation({
    mutationFn: (category) => storageApi.clearCategory(category),
    onSuccess: invalidate,
  })
  const clearAllMut = useMutation({
    mutationFn: storageApi.clearAll,
    onSuccess: invalidate,
  })

  const disk = overview?.disk
  const cats = overview?.categories || []
  const totalMedia = overview?.total || 0

  return (
    <div className="space-y-4">
      {/* Disk usage */}
      <Card className="p-5">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <span className="flex h-10 w-10 items-center justify-center rounded-xl bg-brand-50">
              <span className="icon-[tabler--server-2] size-5 text-brand-600" />
            </span>
            <div>
              <h3 className="font-semibold text-gray-900">Disk Usage</h3>
              <p className="text-xs text-gray-400">{disk?.path || 'storage volume'}</p>
            </div>
          </div>
          <ConfirmButton
            label="Clear ALL media"
            danger
            icon="icon-[tabler--trash]"
            pending={clearAllMut.isPending}
            onConfirm={() => clearAllMut.mutate()}
          />
        </div>

        {disk && (
          <div className="mt-4">
            <div className="mb-1 flex justify-between text-sm">
              <span className="text-gray-500">
                {fmtBytes(disk.used)} used of {fmtBytes(disk.total)}
              </span>
              <span className={`font-medium ${disk.used_percent > 85 ? 'text-red-600' : 'text-gray-700'}`}>
                {disk.used_percent.toFixed(1)}%
              </span>
            </div>
            <div className="h-3 w-full overflow-hidden rounded-full bg-gray-100">
              <div
                className={`h-full rounded-full ${disk.used_percent > 85 ? 'bg-red-500' : 'bg-brand-500'}`}
                style={{ width: `${Math.min(100, disk.used_percent)}%` }}
              />
            </div>
            <div className="mt-2 flex justify-between text-xs text-gray-400">
              <span>Free: {fmtBytes(disk.free)}</span>
              <span>WA media on disk: {fmtBytes(totalMedia)}</span>
            </div>
          </div>
        )}
      </Card>

      {/* Category cards */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        {cats.map((c) => {
          const meta = CAT_META[c.category] || {}
          return (
            <button
              key={c.category}
              onClick={() => setActive(c.category)}
              className={`rounded-2xl bg-white p-4 text-left shadow-sm transition ${
                active === c.category ? 'ring-2 ring-brand-500' : 'hover:shadow'
              }`}
            >
              <div className="flex items-center justify-between">
                <span className={`flex h-9 w-9 items-center justify-center rounded-lg ${meta.bg}`}>
                  <span className={`${meta.icon} size-5 ${meta.color}`} />
                </span>
                <span className="text-xs text-gray-400">{c.files} files</span>
              </div>
              <p className="mt-3 text-sm font-medium text-gray-700">{meta.label}</p>
              <p className="text-lg font-bold text-gray-900">{fmtBytes(c.size)}</p>
            </button>
          )
        })}
      </div>

      {/* File manager */}
      <Card className="overflow-hidden">
        <div className="flex items-center justify-between border-b border-gray-100 px-4 py-3">
          <div className="flex items-center gap-2">
            <span className={`${CAT_META[active]?.icon} size-5 ${CAT_META[active]?.color}`} />
            <h3 className="font-semibold text-gray-900">{CAT_META[active]?.label}</h3>
            <span className="rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-500">{files.length}</span>
          </div>
          {files.length > 0 && (
            <ConfirmButton
              label={`Clear ${CAT_META[active]?.label}`}
              danger
              icon="icon-[tabler--eraser]"
              pending={clearCatMut.isPending}
              onConfirm={() => clearCatMut.mutate(active)}
            />
          )}
        </div>

        <div className="max-h-[55vh] overflow-y-auto">
          {files.length === 0 ? (
            <div className="flex flex-col items-center gap-2 p-10 text-center">
              <span className="icon-[tabler--folder-off] size-8 text-gray-300" />
              <p className="text-sm text-gray-400">No files in this category.</p>
            </div>
          ) : (
            <div className="divide-y divide-gray-50">
              {files.map((f) => (
                <div key={f.name} className="flex items-center gap-3 px-4 py-2.5 hover:bg-gray-50">
                  {/* Thumb / icon */}
                  {isImage(f.name) ? (
                    <img
                      src={mediaUrl(f.path)}
                      alt=""
                      onClick={() => setPreview({ path: f.path, name: f.name })}
                      className="h-10 w-10 flex-none cursor-pointer rounded-lg object-cover"
                    />
                  ) : (
                    <span className="flex h-10 w-10 flex-none items-center justify-center rounded-lg bg-gray-100">
                      <span className="icon-[tabler--file] size-5 text-gray-400" />
                    </span>
                  )}
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-sm font-medium text-gray-700">{f.name}</p>
                    <p className="text-xs text-gray-400">{fmtBytes(f.size)} · {fmtDate(f.mod_time)}</p>
                  </div>
                  <div className="flex flex-none items-center gap-1">
                    {isImage(f.name) && (
                      <button
                        onClick={() => setPreview({ path: f.path, name: f.name })}
                        className="flex h-8 w-8 items-center justify-center rounded-lg text-gray-400 hover:bg-gray-200"
                        title="Preview"
                      >
                        <span className="icon-[tabler--eye] size-4" />
                      </button>
                    )}
                    <button
                      onClick={() => downloadMedia(f.path, f.name)}
                      className="flex h-8 w-8 items-center justify-center rounded-lg text-gray-400 hover:bg-gray-200"
                      title="Download"
                    >
                      <span className="icon-[tabler--download] size-4" />
                    </button>
                    <button
                      onClick={() => delMut.mutate({ category: active, name: f.name })}
                      className="flex h-8 w-8 items-center justify-center rounded-lg text-gray-400 hover:bg-red-50 hover:text-red-600"
                      title="Delete"
                    >
                      <span className="icon-[tabler--trash] size-4" />
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </Card>

      <Lightbox image={preview} onClose={() => setPreview(null)} />
    </div>
  )
}
