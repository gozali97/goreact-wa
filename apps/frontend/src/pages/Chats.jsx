import { useState, useEffect, useRef } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { chatApi, mediaUrl, downloadMedia } from '../lib/api.js'
import EmojiPicker from '../components/EmojiPicker.jsx'
import Lightbox from '../components/Lightbox.jsx'

function formatTime(ts) {
  if (!ts) return ''
  const d = new Date(ts)
  return d.toLocaleString([], {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function initials(text) {
  return (text || '?').trim().charAt(0).toUpperCase()
}

function MessageBubble({ msg, onPreview }) {
  const outgoing = msg.direction === 'outgoing'
  const pending = msg._pending || msg.status === 'pending'
  const DEFAULT_NAMES = {
    image: 'image.jpg',
    video: 'video.mp4',
    audio: 'audio.ogg',
    sticker: 'sticker.webp',
    document: 'document',
  }
  const fileName = msg.file_name || DEFAULT_NAMES[msg.message_type] || 'file'
  // For optimistic image bubbles we have a local blob preview before upload.
  const imageSrc = msg._localPreview || (msg.media_url ? mediaUrl(msg.media_url) : '')
  return (
    <div className={`flex ${outgoing ? 'justify-end' : 'justify-start'}`}>
      <div
        className={`group max-w-[80%] rounded-2xl px-3.5 py-2 text-sm sm:max-w-[70%] ${
          outgoing
            ? 'rounded-br-md bg-brand-500 text-white'
            : 'rounded-bl-md border border-gray-100 bg-white text-gray-800'
        } ${pending ? 'opacity-80' : ''}`}
      >
        {msg.message_type === 'image' && imageSrc && (
          <div className="relative mb-1">
            <img
              src={imageSrc}
              alt="attachment"
              onClick={() => !pending && onPreview({ path: msg.media_url, name: fileName })}
              className={`max-h-60 rounded-lg ${pending ? '' : 'cursor-zoom-in'}`}
            />
            {pending ? (
              <div className="absolute inset-0 flex items-center justify-center rounded-lg bg-black/30">
                <span className="icon-[svg-spinners--ring-resize] size-7 text-white" />
              </div>
            ) : (
              <div className="absolute right-1.5 top-1.5 flex gap-1 opacity-0 transition group-hover:opacity-100">
                <button
                  onClick={() => onPreview({ path: msg.media_url, name: fileName })}
                  className="flex h-7 w-7 items-center justify-center rounded-lg bg-black/50 text-white hover:bg-black/70"
                  aria-label="Preview"
                  title="Preview"
                >
                  <span className="icon-[tabler--eye] size-4" />
                </button>
                <button
                  onClick={() => downloadMedia(msg.media_url, fileName)}
                  className="flex h-7 w-7 items-center justify-center rounded-lg bg-black/50 text-white hover:bg-black/70"
                  aria-label="Download"
                  title="Download"
                >
                  <span className="icon-[tabler--download] size-4" />
                </button>
              </div>
            )}
          </div>
        )}
        {msg.message_type === 'sticker' && msg.media_url && (
          <img
            src={mediaUrl(msg.media_url)}
            alt="sticker"
            className="mb-1 max-h-32 max-w-32"
          />
        )}
        {msg.message_type === 'video' && (
          <div className="relative mb-1">
            {msg.media_url ? (
              <video
                src={mediaUrl(msg.media_url)}
                controls
                className="max-h-72 max-w-full rounded-lg"
              />
            ) : (
              <div className="flex h-40 w-56 items-center justify-center rounded-lg bg-black/20">
                <span className="icon-[svg-spinners--ring-resize] size-7 text-white" />
              </div>
            )}
            {!pending && msg.media_url && (
              <button
                onClick={() => downloadMedia(msg.media_url, fileName)}
                className="absolute right-1.5 top-1.5 flex h-7 w-7 items-center justify-center rounded-lg bg-black/50 text-white opacity-0 transition hover:bg-black/70 group-hover:opacity-100"
                aria-label="Download"
                title="Download"
              >
                <span className="icon-[tabler--download] size-4" />
              </button>
            )}
          </div>
        )}
        {msg.message_type === 'audio' && msg.media_url && (
          <audio src={mediaUrl(msg.media_url)} controls className="mb-1 max-w-full" />
        )}
        {msg.message_type === 'document' && (
          <div
            className={`mb-1 flex items-center gap-2 rounded-lg p-2 ${
              outgoing ? 'bg-white/15' : 'bg-gray-50'
            }`}
          >
            <span
              className={`flex h-9 w-9 flex-none items-center justify-center rounded-lg ${
                outgoing ? 'bg-white/20' : 'bg-brand-50'
              }`}
            >
              {pending ? (
                <span className="icon-[svg-spinners--ring-resize] size-5 text-white" />
              ) : (
                <span
                  className={`icon-[tabler--file] size-5 ${outgoing ? 'text-white' : 'text-brand-600'}`}
                />
              )}
            </span>
            <span className="min-w-0 flex-1 truncate">{fileName}</span>
            {!pending && msg.media_url && (
              <button
                onClick={() => downloadMedia(msg.media_url, fileName)}
                className={`flex h-7 w-7 flex-none items-center justify-center rounded-lg ${
                  outgoing ? 'hover:bg-white/20' : 'hover:bg-gray-200'
                }`}
                aria-label="Download"
                title="Download"
              >
                <span className="icon-[tabler--download] size-4" />
              </button>
            )}
          </div>
        )}
        {msg.content && <p className="whitespace-pre-wrap break-words">{msg.content}</p>}
        <div className={`mt-1 flex items-center justify-end gap-1 text-[10px] ${outgoing ? 'text-brand-100' : 'text-gray-400'}`}>
          {pending ? (
            <>
              <span className="icon-[svg-spinners--ring-resize] size-3" />
              <span>Sending…</span>
            </>
          ) : (
            <span>
              {formatTime(msg.sent_at || msg.created_at)}
              {outgoing && msg.status ? ` · ${msg.status}` : ''}
            </span>
          )}
        </div>
      </div>
    </div>
  )
}

export default function Chats() {
  const queryClient = useQueryClient()
  const [activeId, setActiveId] = useState(null)
  const [text, setText] = useState('')
  const [showEmoji, setShowEmoji] = useState(false)
  const [attachment, setAttachment] = useState(null) // { file, url, isImage }
  const [preview, setPreview] = useState(null) // { path, name } for lightbox
  const [showNew, setShowNew] = useState(false)
  const [newPhone, setNewPhone] = useState('')
  const [newMessage, setNewMessage] = useState('')
  const [newError, setNewError] = useState('')
  const bottomRef = useRef(null)
  const inputRef = useRef(null)
  const fileInputRef = useRef(null)
  const imageInputRef = useRef(null)

  const { data: convData } = useQuery({
    queryKey: ['conversations'],
    queryFn: chatApi.list,
    refetchInterval: 15000,
  })
  const conversations = convData?.conversations || []

  const { data: detail } = useQuery({
    queryKey: ['chat-detail', activeId],
    queryFn: () => chatApi.detail(activeId),
    enabled: !!activeId,
    refetchInterval: 10000,
  })
  const messages = detail?.messages || []

  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: ['chat-detail', activeId] })
    queryClient.invalidateQueries({ queryKey: ['conversations'] })
  }

  // Append a temporary "pending" message to the chat-detail cache so the UI
  // updates instantly and shows a sending indicator.
  const addPendingMessage = (msg) => {
    queryClient.setQueryData(['chat-detail', activeId], (old) => {
      if (!old) return old
      return { ...old, messages: [...(old.messages || []), msg] }
    })
  }

  const replyMut = useMutation({
    mutationFn: ({ id, message }) => chatApi.reply(id, message),
    onMutate: ({ message }) => {
      const tempId = `tmp-${Date.now()}`
      addPendingMessage({
        id: tempId,
        direction: 'outgoing',
        message_type: 'text',
        content: message,
        status: 'pending',
        created_at: new Date().toISOString(),
        _pending: true,
      })
      setText('')
      return { tempId }
    },
    onSettled: () => invalidate(),
  })

  const mediaMut = useMutation({
    mutationFn: ({ id, formData }) => chatApi.replyMedia(id, formData),
    onMutate: ({ caption, previewURL, isImage, fileName }) => {
      const tempId = `tmp-${Date.now()}`
      addPendingMessage({
        id: tempId,
        direction: 'outgoing',
        message_type: isImage ? 'image' : 'document',
        content: caption || '',
        media_url: '', // local blob shown via _localPreview instead
        file_name: fileName,
        status: 'pending',
        created_at: new Date().toISOString(),
        _pending: true,
        _localPreview: isImage ? previewURL : '',
      })
      clearAttachment()
      setText('')
      return { tempId }
    },
    onSettled: () => invalidate(),
  })

  const startMut = useMutation({
    mutationFn: ({ phone, message }) => chatApi.start(phone, message),
    onSuccess: (res) => {
      setShowNew(false)
      setNewPhone('')
      setNewMessage('')
      setNewError('')
      queryClient.invalidateQueries({ queryKey: ['conversations'] })
      if (res?.conversation_id) setActiveId(res.conversation_id)
    },
    onError: (e) => setNewError(e?.response?.data?.error || e.message),
  })

  const handleStartChat = (e) => {
    e.preventDefault()
    setNewError('')
    if (!newPhone.trim() || !newMessage.trim()) {
      setNewError('Phone and message are required.')
      return
    }
    startMut.mutate({ phone: newPhone.trim(), message: newMessage.trim() })
  }

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages.length])

  // Reset composer state when switching conversations.
  useEffect(() => {
    setShowEmoji(false)
    clearAttachment()
    setText('')
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeId])

  const activeConv = conversations.find((c) => c.id === activeId)
  const sending = replyMut.isPending || mediaMut.isPending

  function clearAttachment() {
    // Note: we intentionally don't revoke the object URL here because an
    // optimistic "sending" bubble may still be displaying it. It will be GC'd
    // when the page unloads / conversation switches.
    setAttachment(null)
    if (fileInputRef.current) fileInputRef.current.value = ''
    if (imageInputRef.current) imageInputRef.current.value = ''
  }

  function pickFile(file) {
    if (!file) return
    const isImage = file.type.startsWith('image/')
    setAttachment({
      file,
      isImage,
      url: isImage ? URL.createObjectURL(file) : '',
    })
    setShowEmoji(false)
  }

  function insertEmoji(emoji) {
    const el = inputRef.current
    if (!el) {
      setText((t) => t + emoji)
      return
    }
    const start = el.selectionStart ?? text.length
    const end = el.selectionEnd ?? text.length
    const next = text.slice(0, start) + emoji + text.slice(end)
    setText(next)
    // Restore caret after the inserted emoji.
    requestAnimationFrame(() => {
      el.focus()
      const pos = start + emoji.length
      el.setSelectionRange(pos, pos)
    })
  }

  const handleSend = (e) => {
    e.preventDefault()
    if (!activeId || sending) return

    if (attachment) {
      const fd = new FormData()
      fd.append('file', attachment.file)
      if (text.trim()) fd.append('caption', text.trim())
      mediaMut.mutate({
        id: activeId,
        formData: fd,
        caption: text.trim(),
        isImage: attachment.isImage,
        fileName: attachment.file.name,
        previewURL: attachment.url,
      })
      return
    }
    if (!text.trim()) return
    replyMut.mutate({ id: activeId, message: text.trim() })
  }

  return (
    <div className="flex h-[calc(100vh-7.5rem)] gap-4 sm:h-[calc(100vh-8.5rem)]">
      {/* Conversation list */}
      <div
        className={`flex w-full flex-col overflow-hidden rounded-2xl bg-white md:w-80 md:flex-none ${
          activeId ? 'hidden md:flex' : 'flex'
        }`}
      >
        <div className="flex items-center justify-between border-b border-gray-100 px-4 py-3.5">
          <h2 className="font-semibold text-gray-900">Conversations</h2>
          <div className="flex items-center gap-2">
            <span className="rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-500">
              {conversations.length}
            </span>
            <button
              onClick={() => setShowNew(true)}
              className="flex h-8 w-8 items-center justify-center rounded-lg bg-brand-500 text-white hover:bg-brand-600"
              title="New chat"
              aria-label="New chat"
            >
              <span className="icon-[tabler--edit] size-4" />
            </button>
          </div>
        </div>
        <div className="flex-1 overflow-y-auto">
          {conversations.length === 0 && (
            <div className="flex flex-col items-center gap-2 p-8 text-center">
              <span className="icon-[tabler--message-off] size-8 text-gray-300" />
              <p className="text-sm text-gray-400">No conversations yet.</p>
            </div>
          )}
          {conversations.map((conv) => (
            <button
              key={conv.id}
              onClick={() => setActiveId(conv.id)}
              className={`flex w-full items-start gap-3 border-b border-gray-50 px-4 py-3 text-left transition hover:bg-gray-50 ${
                activeId === conv.id ? 'bg-brand-50' : ''
              }`}
            >
              <div className="flex h-10 w-10 flex-none items-center justify-center rounded-full bg-gradient-to-br from-brand-400 to-brand-600 text-sm font-semibold text-white">
                {initials(conv.contact?.name || conv.contact?.phone)}
              </div>
              <div className="min-w-0 flex-1">
                <div className="flex items-center justify-between gap-2">
                  <span className="truncate text-sm font-medium text-gray-900">
                    {conv.contact?.name || conv.contact?.phone}
                  </span>
                  {conv.unread_count > 0 && (
                    <span className="flex h-5 min-w-5 items-center justify-center rounded-full bg-brand-500 px-1.5 text-[10px] font-semibold text-white">
                      {conv.unread_count}
                    </span>
                  )}
                </div>
                <p className="truncate text-xs text-gray-500">{conv.last_message || '—'}</p>
              </div>
            </button>
          ))}
        </div>
      </div>

      {/* Chat detail */}
      <div
        className={`flex-1 flex-col overflow-hidden rounded-2xl bg-gray-50 ${
          activeId ? 'flex' : 'hidden md:flex'
        }`}
      >
        {activeId ? (
          <>
            <div className="flex items-center gap-3 border-b border-gray-200 bg-white px-4 py-3">
              <button
                onClick={() => setActiveId(null)}
                className="flex h-9 w-9 items-center justify-center rounded-lg text-gray-500 hover:bg-gray-100 md:hidden"
                aria-label="Back"
              >
                <span className="icon-[tabler--arrow-left] size-5" />
              </button>
              <div className="flex h-9 w-9 flex-none items-center justify-center rounded-full bg-gradient-to-br from-brand-400 to-brand-600 text-sm font-semibold text-white">
                {initials(activeConv?.contact?.name || activeConv?.contact?.phone)}
              </div>
              <div className="min-w-0">
                <p className="truncate font-medium text-gray-900">
                  {activeConv?.contact?.name || activeConv?.contact?.phone}
                </p>
                <p className="truncate text-xs text-gray-400">{activeConv?.contact?.phone}</p>
              </div>
            </div>

            <div className="flex-1 space-y-2 overflow-y-auto p-4">
              {messages.map((m) => (
                <MessageBubble key={m.id} msg={m} onPreview={setPreview} />
              ))}
              <div ref={bottomRef} />
            </div>

            <form onSubmit={handleSend} className="border-t border-gray-200 bg-white p-3">
              {/* Attachment preview */}
              {attachment && (
                <div className="mb-2 flex items-center gap-3 rounded-xl bg-gray-50 p-2">
                  {attachment.isImage ? (
                    <img
                      src={attachment.url}
                      alt="preview"
                      className="h-12 w-12 rounded-lg object-cover"
                    />
                  ) : (
                    <span className="flex h-12 w-12 items-center justify-center rounded-lg bg-brand-50">
                      <span className="icon-[tabler--file] size-6 text-brand-600" />
                    </span>
                  )}
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-sm font-medium text-gray-800">
                      {attachment.file.name}
                    </p>
                    <p className="text-xs text-gray-400">
                      {(attachment.file.size / 1024).toFixed(0)} KB ·{' '}
                      {attachment.isImage ? 'Photo' : 'Document'}
                    </p>
                  </div>
                  <button
                    type="button"
                    onClick={clearAttachment}
                    className="flex h-8 w-8 items-center justify-center rounded-lg text-gray-400 hover:bg-gray-200"
                    aria-label="Remove attachment"
                  >
                    <span className="icon-[tabler--x] size-4" />
                  </button>
                </div>
              )}

              <div className="relative flex items-end gap-2">
                {showEmoji && (
                  <EmojiPicker
                    onSelect={insertEmoji}
                    onClose={() => setShowEmoji(false)}
                  />
                )}

                {/* Emoji */}
                <button
                  type="button"
                  onClick={() => setShowEmoji((v) => !v)}
                  className={`flex h-11 w-10 flex-none items-center justify-center rounded-xl hover:bg-gray-100 ${
                    showEmoji ? 'text-brand-600' : 'text-gray-500'
                  }`}
                  aria-label="Emoji"
                >
                  <span className="icon-[tabler--mood-smile] size-5" />
                </button>

                {/* Attach photo */}
                <button
                  type="button"
                  onClick={() => imageInputRef.current?.click()}
                  className="flex h-11 w-10 flex-none items-center justify-center rounded-xl text-gray-500 hover:bg-gray-100"
                  aria-label="Send photo"
                >
                  <span className="icon-[tabler--photo] size-5" />
                </button>

                {/* Attach file */}
                <button
                  type="button"
                  onClick={() => fileInputRef.current?.click()}
                  className="flex h-11 w-10 flex-none items-center justify-center rounded-xl text-gray-500 hover:bg-gray-100"
                  aria-label="Attach file"
                >
                  <span className="icon-[tabler--paperclip] size-5" />
                </button>

                <input
                  ref={inputRef}
                  value={text}
                  onChange={(e) => setText(e.target.value)}
                  placeholder={attachment ? 'Add a caption…' : 'Type a message…'}
                  className="flex-1 rounded-xl border border-gray-200 px-4 py-2.5 text-sm focus:border-brand-500 focus:outline-none"
                />

                <button
                  type="submit"
                  disabled={sending || (!text.trim() && !attachment)}
                  className="flex h-11 w-11 flex-none items-center justify-center rounded-xl bg-brand-500 text-white hover:bg-brand-600 disabled:opacity-50"
                  aria-label="Send"
                >
                  {sending ? (
                    <span className="icon-[svg-spinners--3-dots-move] size-5" />
                  ) : (
                    <span className="icon-[tabler--send] size-5" />
                  )}
                </button>
              </div>

              {/* Hidden file inputs */}
              <input
                ref={imageInputRef}
                type="file"
                accept="image/*"
                className="hidden"
                onChange={(e) => pickFile(e.target.files?.[0])}
              />
              <input
                ref={fileInputRef}
                type="file"
                className="hidden"
                onChange={(e) => pickFile(e.target.files?.[0])}
              />
            </form>
          </>
        ) : (
          <div className="flex flex-1 flex-col items-center justify-center gap-2 text-gray-400">
            <span className="icon-[tabler--messages] size-10 text-gray-300" />
            <p className="text-sm">Select a conversation to view messages</p>
          </div>
        )}
      </div>

      {/* Full-size image preview */}
      <Lightbox image={preview} onClose={() => setPreview(null)} />

      {/* New chat modal */}
      {showNew && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
          onClick={() => setShowNew(false)}
        >
          <div className="w-full max-w-sm rounded-2xl bg-white p-5 shadow-xl" onClick={(e) => e.stopPropagation()}>
            <div className="mb-4 flex items-center gap-3">
              <span className="flex h-10 w-10 items-center justify-center rounded-xl bg-brand-50">
                <span className="icon-[tabler--message-plus] size-5 text-brand-600" />
              </span>
              <div>
                <h3 className="font-semibold text-gray-900">New chat</h3>
                <p className="text-xs text-gray-400">Start a conversation with a number</p>
              </div>
            </div>
            <form onSubmit={handleStartChat} className="space-y-3">
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700">Phone number</label>
                <input
                  value={newPhone}
                  onChange={(e) => setNewPhone(e.target.value)}
                  autoFocus
                  placeholder="628123456789"
                  className="w-full rounded-xl border border-gray-200 px-3 py-2.5 text-sm focus:border-brand-500 focus:outline-none"
                />
              </div>
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700">Message</label>
                <textarea
                  value={newMessage}
                  onChange={(e) => setNewMessage(e.target.value)}
                  rows={3}
                  placeholder="Hello!"
                  className="w-full rounded-xl border border-gray-200 px-3 py-2.5 text-sm focus:border-brand-500 focus:outline-none"
                />
              </div>
              {newError && (
                <div className="flex items-center gap-2 rounded-lg bg-red-50 px-3 py-2 text-sm text-red-600">
                  <span className="icon-[tabler--alert-circle] size-4" />
                  {newError}
                </div>
              )}
              <div className="flex justify-end gap-2 pt-1">
                <button
                  type="button"
                  onClick={() => setShowNew(false)}
                  className="rounded-xl px-4 py-2 text-sm font-medium text-gray-500 hover:bg-gray-100"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={startMut.isPending}
                  className="inline-flex items-center gap-2 rounded-xl bg-brand-500 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-600 disabled:opacity-50"
                >
                  {startMut.isPending ? (
                    <span className="icon-[svg-spinners--3-dots-move] size-4" />
                  ) : (
                    <span className="icon-[tabler--send] size-4" />
                  )}
                  Start chat
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
