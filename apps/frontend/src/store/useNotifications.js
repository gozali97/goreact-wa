import { create } from 'zustand'

const MAX = 50

let seq = 0
function nextId() {
  seq += 1
  return `${Date.now()}-${seq}`
}

// In-memory notifications feed populated from WebSocket events. Holds the most
// recent MAX items plus an unread counter for the navbar bell.
export const useNotifications = create((set, get) => ({
  items: [], // { id, type, title, body, to, time, read }
  unread: 0,

  add: (n) =>
    set((s) => {
      const item = {
        id: nextId(),
        time: Date.now(),
        read: false,
        ...n,
      }
      const items = [item, ...s.items].slice(0, MAX)
      return { items, unread: s.unread + 1 }
    }),

  markAllRead: () =>
    set((s) => ({
      items: s.items.map((i) => ({ ...i, read: true })),
      unread: 0,
    })),

  clear: () => set({ items: [], unread: 0 }),
}))
