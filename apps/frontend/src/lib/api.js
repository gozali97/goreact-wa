import axios from 'axios'
import { useAuthStore } from '../store/useAuthStore.js'

// Axios instance. Credentials are sent as an Authorization header built from
// the token in the auth store (set via the login screen).
const api = axios.create({
  baseURL: '/api',
})

// Attach the Authorization header on every request.
api.interceptors.request.use((config) => {
  const token = useAuthStore.getState().token
  if (token) {
    config.headers.Authorization = `Basic ${token}`
  }
  return config
})

// On 401, drop the stored credentials so the app returns to the login screen.
api.interceptors.response.use(
  (res) => res,
  (error) => {
    if (error?.response?.status === 401) {
      useAuthStore.getState().logout()
    }
    return Promise.reject(error)
  }
)

// ─── Auth ─────────────────────────────────────────────────
export const authApi = {
  check: (token) =>
    api
      .get('/auth/check', { headers: { Authorization: `Basic ${token}` } })
      .then((r) => r.data),
}

// ─── Device ───────────────────────────────────────────────
export const deviceApi = {
  connect: () => api.post('/device/connect').then((r) => r.data),
  qr: () => api.get('/device/qr').then((r) => r.data),
  status: () => api.get('/device/status').then((r) => r.data),
  logout: () => api.post('/device/logout').then((r) => r.data),
  getApiKey: () => api.get('/device/api-key').then((r) => r.data),
  generateApiKey: () => api.post('/device/api-key/generate').then((r) => r.data),
}

// ─── Messages ─────────────────────────────────────────────
export const messageApi = {
  send: (phone, message) =>
    api.post('/messages/send', { phone, message }).then((r) => r.data),
  sendImage: (formData) =>
    api
      .post('/messages/send-image', formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
      })
      .then((r) => r.data),
  sendFile: (formData) =>
    api
      .post('/messages/send-file', formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
      })
      .then((r) => r.data),
}

// ─── Conversations / Chats ────────────────────────────────
export const chatApi = {
  list: () => api.get('/conversations').then((r) => r.data),
  detail: (id) => api.get(`/conversations/${id}`).then((r) => r.data),
  reply: (id, message) =>
    api.post(`/conversations/${id}/reply`, { message }).then((r) => r.data),
  replyMedia: (id, formData) =>
    api
      .post(`/conversations/${id}/reply-media`, formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
      })
      .then((r) => r.data),
  start: (phone, message) =>
    api.post('/chats/start', { phone, message }).then((r) => r.data),
}

// ─── Contacts ─────────────────────────────────────────────
export const contactApi = {
  list: (search) =>
    api.get('/contacts', { params: { search } }).then((r) => r.data),
  detail: (id) => api.get(`/contacts/${id}`).then((r) => r.data),
}

// ─── Broadcast ────────────────────────────────────────────
export const broadcastApi = {
  create: (payload) => api.post('/broadcasts', payload).then((r) => r.data),
  list: () => api.get('/broadcasts').then((r) => r.data),
  detail: (id) => api.get(`/broadcasts/${id}`).then((r) => r.data),
}

// ─── Dashboard ────────────────────────────────────────────
export const dashboardApi = {
  stats: () => api.get('/dashboard/stats').then((r) => r.data),
}

// ─── Storage admin (disk monitoring + file manager) ──────
export const storageApi = {
  overview: () => api.get('/storage-admin/overview').then((r) => r.data),
  files: (category) => api.get(`/storage-admin/files/${category}`).then((r) => r.data),
  deleteFile: (category, name) =>
    api.delete(`/storage-admin/files/${category}`, { params: { name } }).then((r) => r.data),
  clearCategory: (category) =>
    api.delete(`/storage-admin/category/${category}`).then((r) => r.data),
  clearAll: () => api.delete('/storage-admin/all').then((r) => r.data),
}

// ─── Logs ─────────────────────────────────────────────────
export const logApi = {
  days: () => api.get('/logs/days').then((r) => r.data),
  entries: (params) => api.get('/logs', { params }).then((r) => r.data),
  downloadUrl: (day) => {
    const token = useAuthStore.getState().token
    const qs = new URLSearchParams()
    if (day) qs.set('day', day)
    if (token) qs.set('access_token', token)
    return `/api/logs/download?${qs.toString()}`
  },
}

// ─── Generic runner (used by the interactive API docs) ────
// Sends an arbitrary request through the authed client and returns a normalized
// result with status, duration and parsed body (never throws on HTTP errors).
export async function runRequest({ method, path, query, body, formData, contentType }) {
  const config = {
    method,
    url: path, // relative to baseURL '/api'
    params: query,
    validateStatus: () => true, // we want to display 4xx/5xx too
  }
  if (formData) {
    config.data = formData
    config.headers = { 'Content-Type': 'multipart/form-data' }
  } else if (body !== undefined) {
    config.data = body
  }
  if (contentType && !formData) {
    config.headers = { ...(config.headers || {}), 'Content-Type': contentType }
  }

  const started = performance.now()
  const res = await api.request(config)
  const ms = Math.round(performance.now() - started)
  return { status: res.status, statusText: res.statusText, durationMs: ms, data: res.data }
}

// mediaUrl appends the auth token to a /storage path so <img>/<a> tags (which
// can't send an Authorization header) can load auth-protected media.
export function mediaUrl(path) {
  if (!path) return ''
  const token = useAuthStore.getState().token
  if (!token) return path
  const sep = path.includes('?') ? '&' : '?'
  return `${path}${sep}access_token=${encodeURIComponent(token)}`
}

// downloadMedia fetches an auth-protected media file as a blob and triggers a
// browser download with the given filename (falls back to the path's basename).
export async function downloadMedia(path, fileName) {
  if (!path) return
  const res = await api.get(path, {
    baseURL: '',
    responseType: 'blob',
  })
  const blobUrl = URL.createObjectURL(res.data)
  const a = document.createElement('a')
  a.href = blobUrl
  a.download = fileName || path.split('/').pop() || 'download'
  document.body.appendChild(a)
  a.click()
  a.remove()
  // Release the object URL after the download starts.
  setTimeout(() => URL.revokeObjectURL(blobUrl), 4000)
}

export default api
