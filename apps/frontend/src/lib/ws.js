// Lightweight WebSocket client with auto-reconnect and a tiny pub/sub layer.

import { useAuthStore } from '../store/useAuthStore.js'

function buildURL() {
  const proto = window.location.protocol === 'https:' ? 'wss' : 'ws'
  const token = useAuthStore.getState().token
  const qs = token ? `?access_token=${encodeURIComponent(token)}` : ''
  return `${proto}://${window.location.host}/ws${qs}`
}

class WSClient {
  constructor() {
    this.socket = null
    this.listeners = new Map() // type -> Set<fn>
    this.reconnectTimer = null
    this.shouldRun = false
  }

  connect() {
    this.shouldRun = true
    this._open()
  }

  _open() {
    if (this.socket && this.socket.readyState === WebSocket.OPEN) return
    try {
      this.socket = new WebSocket(buildURL())
    } catch {
      this._scheduleReconnect()
      return
    }

    this.socket.onmessage = (ev) => {
      let msg
      try {
        msg = JSON.parse(ev.data)
      } catch {
        return
      }
      this._emit(msg.type, msg.data)
      this._emit('*', msg)
    }

    this.socket.onclose = () => {
      if (this.shouldRun) this._scheduleReconnect()
    }

    this.socket.onerror = () => {
      if (this.socket) this.socket.close()
    }
  }

  _scheduleReconnect() {
    if (this.reconnectTimer) return
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null
      this._open()
    }, 3000)
  }

  on(type, fn) {
    if (!this.listeners.has(type)) this.listeners.set(type, new Set())
    this.listeners.get(type).add(fn)
    return () => this.off(type, fn)
  }

  off(type, fn) {
    const set = this.listeners.get(type)
    if (set) set.delete(fn)
  }

  _emit(type, data) {
    const set = this.listeners.get(type)
    if (set) set.forEach((fn) => fn(data))
  }

  close() {
    this.shouldRun = false
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
    if (this.socket) this.socket.close()
  }
}

const wsClient = new WSClient()
export default wsClient
