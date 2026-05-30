import { useEffect, useRef, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import QRCode from 'qrcode'
import { deviceApi } from '../lib/api.js'
import { useAppStore } from '../store/useAppStore.js'
import Card from '../components/Card.jsx'
import StatusBadge from '../components/StatusBadge.jsx'
import ApiKeyCard from '../components/ApiKeyCard.jsx'

export default function Device() {
  const queryClient = useQueryClient()
  const canvasRef = useRef(null)
  const [error, setError] = useState('')

  const qrCode = useAppStore((s) => s.qrCode)
  const deviceStatus = useAppStore((s) => s.deviceStatus)
  const setQrCode = useAppStore((s) => s.setQrCode)
  const setDeviceStatus = useAppStore((s) => s.setDeviceStatus)
  const setQrPaired = useAppStore((s) => s.setQrPaired)

  const { data: status } = useQuery({
    queryKey: ['device-status'],
    queryFn: deviceApi.status,
    refetchInterval: 10000,
  })

  const connected = deviceStatus === 'connected' || status?.connected

  const connectMut = useMutation({
    mutationFn: deviceApi.connect,
    onMutate: () => {
      setError('')
      setDeviceStatus('connecting')
    },
    onError: (e) => setError(e?.response?.data?.error || e.message),
  })

  const logoutMut = useMutation({
    mutationFn: deviceApi.logout,
    onSuccess: () => {
      setQrCode('')
      setQrPaired(false)
      queryClient.invalidateQueries({ queryKey: ['device-status'] })
    },
  })

  // Poll the QR endpoint as a fallback for the WebSocket push, so the code
  // always appears even if a WS event is missed. Active only while we're
  // trying to connect and not yet connected / not yet showing a code.
  const wantQr = !connected && (deviceStatus === 'connecting' || connectMut.isPending)
  useQuery({
    queryKey: ['device-qr'],
    queryFn: async () => {
      const data = await deviceApi.qr()
      if (data?.code) setQrCode(data.code)
      if (data?.status) setDeviceStatus(data.status)
      return data
    },
    enabled: wantQr,
    refetchInterval: 2000,
  })

  // Render the QR string to canvas whenever it changes.
  useEffect(() => {
    if (qrCode && !connected && canvasRef.current) {
      QRCode.toCanvas(canvasRef.current, qrCode, { width: 240, margin: 1 }, (err) => {
        if (err) console.error(err)
      })
    }
  }, [qrCode, connected])

  const showQr = qrCode && !connected

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
      {/* Connection state */}
      <Card className="p-6 lg:col-span-2">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <p className="text-sm text-gray-400">Connection Status</p>
            <h3 className="mt-1 text-2xl font-bold text-gray-900">
              {connected ? 'Connected' : showQr ? 'Scan to Connect' : 'Disconnected'}
            </h3>
          </div>
          <div className="flex items-center gap-2">
            <StatusBadge status={deviceStatus} />
            {!connected ? (
              <button
                onClick={() => connectMut.mutate()}
                disabled={connectMut.isPending}
                className="inline-flex items-center gap-2 rounded-xl bg-brand-500 px-4 py-2.5 text-sm font-semibold text-white hover:bg-brand-600 disabled:opacity-50"
              >
                <span className="icon-[tabler--plug-connected] size-4" />
                {connectMut.isPending ? 'Connecting…' : showQr ? 'Refresh QR' : 'Connect'}
              </button>
            ) : (
              <button
                onClick={() => logoutMut.mutate()}
                disabled={logoutMut.isPending}
                className="inline-flex items-center gap-2 rounded-xl border border-red-200 px-4 py-2.5 text-sm font-semibold text-red-600 hover:bg-red-50 disabled:opacity-50"
              >
                <span className="icon-[tabler--logout] size-4" />
                Logout
              </button>
            )}
          </div>
        </div>

        {error && (
          <div className="mt-4 flex items-center gap-2 rounded-xl bg-red-50 px-4 py-3 text-sm text-red-600">
            <span className="icon-[tabler--alert-circle] size-4" />
            {error}
          </div>
        )}

        <div className="mt-6">
          {connected ? (
            <div className="flex flex-col items-center gap-2 rounded-2xl bg-brand-50 p-8 text-center">
              <span className="icon-[tabler--circle-check] size-12 text-brand-500" />
              <p className="text-lg font-semibold text-brand-700">WhatsApp is connected</p>
              {status?.phone && (
                <p className="text-sm text-gray-600">
                  {status.push_name ? `${status.push_name} · ` : ''}
                  {status.phone}
                </p>
              )}
            </div>
          ) : showQr ? (
            <div className="flex flex-col items-center gap-4 py-2">
              <div className="rounded-2xl border border-gray-100 bg-white p-4 shadow-sm">
                <canvas ref={canvasRef} />
              </div>
              <p className="max-w-sm text-center text-sm text-gray-500">
                Open WhatsApp → Linked Devices → Link a Device, then scan this code.
              </p>
            </div>
          ) : deviceStatus === 'connecting' || connectMut.isPending ? (
            <div className="flex flex-col items-center gap-3 rounded-2xl bg-gray-50 p-10 text-center">
              <span className="icon-[svg-spinners--3-dots-move] size-10 text-brand-500" />
              <p className="text-sm text-gray-500">Generating QR code…</p>
            </div>
          ) : (
            <div className="flex flex-col items-center gap-3 rounded-2xl bg-gray-50 p-10 text-center">
              <span className="icon-[tabler--qrcode] size-12 text-gray-300" />
              <p className="text-sm text-gray-500">
                Click <span className="font-semibold text-gray-700">Connect</span> to
                generate a QR code and link your WhatsApp account.
              </p>
            </div>
          )}
        </div>
      </Card>

      {/* How it works */}
      <Card className="p-6">
        <h3 className="font-semibold text-gray-900">How it works</h3>
        <ol className="mt-4 space-y-4">
          {[
            { icon: 'icon-[tabler--plug-connected]', text: 'Click Connect to start a session.' },
            { icon: 'icon-[tabler--qrcode]', text: 'A QR code is generated by the server.' },
            { icon: 'icon-[tabler--device-mobile]', text: 'Scan it from WhatsApp on your phone.' },
            { icon: 'icon-[tabler--refresh]', text: 'The session reconnects automatically.' },
          ].map((s, i) => (
            <li key={i} className="flex items-start gap-3">
              <span className="flex h-8 w-8 flex-none items-center justify-center rounded-lg bg-brand-50">
                <span className={`${s.icon} size-4 text-brand-600`} />
              </span>
              <p className="text-sm text-gray-600">{s.text}</p>
            </li>
          ))}
        </ol>
      </Card>
      </div>

      {/* API key — shown once connected */}
      {connected && <ApiKeyCard />}
    </div>
  )
}
