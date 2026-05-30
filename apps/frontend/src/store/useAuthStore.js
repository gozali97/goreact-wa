import { create } from 'zustand'

const STORAGE_KEY = 'wa_proxy_auth'

function loadToken() {
  try {
    return localStorage.getItem(STORAGE_KEY) || ''
  } catch {
    return ''
  }
}

// Auth state: holds a base64("user:pass") token used for the Authorization
// header (REST) and the access_token query param (WebSocket). Persisted in
// localStorage so a refresh keeps the session.
export const useAuthStore = create((set) => ({
  token: loadToken(),
  isAuthed: !!loadToken(),

  setToken: (token) => {
    try {
      if (token) localStorage.setItem(STORAGE_KEY, token)
      else localStorage.removeItem(STORAGE_KEY)
    } catch {
      // ignore storage errors (private mode, etc.)
    }
    set({ token, isAuthed: !!token })
  },

  logout: () => {
    try {
      localStorage.removeItem(STORAGE_KEY)
    } catch {
      // ignore
    }
    set({ token: '', isAuthed: false })
  },
}))

// Encode credentials to the base64 token used everywhere.
export function encodeToken(username, password) {
  return btoa(`${username}:${password}`)
}
