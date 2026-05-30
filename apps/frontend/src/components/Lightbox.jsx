import { useEffect } from 'react'
import { mediaUrl, downloadMedia } from '../lib/api.js'

// Full-screen image preview with a download button. `image` is { path, name }.
export default function Lightbox({ image, onClose }) {
  useEffect(() => {
    const onKey = (e) => {
      if (e.key === 'Escape') onClose?.()
    }
    document.addEventListener('keydown', onKey)
    // Lock background scroll while open.
    const prev = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => {
      document.removeEventListener('keydown', onKey)
      document.body.style.overflow = prev
    }
  }, [onClose])

  if (!image) return null

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 p-4"
      onClick={onClose}
    >
      {/* Toolbar */}
      <div className="absolute right-4 top-4 flex items-center gap-2">
        <button
          onClick={(e) => {
            e.stopPropagation()
            downloadMedia(image.path, image.name)
          }}
          className="flex h-10 w-10 items-center justify-center rounded-xl bg-white/10 text-white hover:bg-white/20"
          aria-label="Download"
          title="Download"
        >
          <span className="icon-[tabler--download] size-5" />
        </button>
        <button
          onClick={onClose}
          className="flex h-10 w-10 items-center justify-center rounded-xl bg-white/10 text-white hover:bg-white/20"
          aria-label="Close"
          title="Close (Esc)"
        >
          <span className="icon-[tabler--x] size-5" />
        </button>
      </div>

      <img
        src={mediaUrl(image.path)}
        alt={image.name || 'preview'}
        onClick={(e) => e.stopPropagation()}
        className="max-h-[88vh] max-w-[92vw] rounded-xl object-contain shadow-2xl"
      />
    </div>
  )
}
