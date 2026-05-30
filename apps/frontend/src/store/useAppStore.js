import { create } from 'zustand'

// Global app state: device connection status and the latest QR code.
export const useAppStore = create((set) => ({
  deviceStatus: 'unknown',
  qrCode: '',
  qrPaired: false,

  setDeviceStatus: (status) => set({ deviceStatus: status }),
  setQrCode: (code) => set({ qrCode: code }),
  setQrPaired: (paired) => set({ qrPaired: paired }),
  reset: () => set({ deviceStatus: 'unknown', qrCode: '', qrPaired: false }),
}))
