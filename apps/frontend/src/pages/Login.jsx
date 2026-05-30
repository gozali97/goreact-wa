import { useState } from 'react'
import { authApi } from '../lib/api.js'
import { useAuthStore, encodeToken } from '../store/useAuthStore.js'

export default function Login() {
  const setToken = useAuthStore((s) => s.setToken)
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (e) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    const token = encodeToken(username, password)
    try {
      await authApi.check(token)
      setToken(token)
    } catch (err) {
      if (err?.response?.status === 401) {
        setError('Invalid username or password.')
      } else {
        setError(err?.message || 'Unable to sign in.')
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex min-h-full items-center justify-center bg-canvas p-4">
      <div className="w-full max-w-sm rounded-2xl bg-white p-8 shadow-xl">
        <div className="mb-6 flex flex-col items-center text-center">
          <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-brand-500 text-white">
            <span className="icon-[tabler--brand-whatsapp] size-7" />
          </div>
          <h1 className="mt-3 text-xl font-bold text-gray-900">WA Proxy</h1>
          <p className="text-sm text-gray-400">Sign in to your dashboard</p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="mb-1 block text-sm font-medium text-gray-700">Username</label>
            <div className="relative">
              <span className="icon-[tabler--user] absolute left-3 top-1/2 size-4 -translate-y-1/2 text-gray-400" />
              <input
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                autoFocus
                autoComplete="username"
                className="w-full rounded-xl border border-gray-200 py-2.5 pl-9 pr-3 text-sm focus:border-brand-500 focus:outline-none"
                placeholder="admin"
              />
            </div>
          </div>

          <div>
            <label className="mb-1 block text-sm font-medium text-gray-700">Password</label>
            <div className="relative">
              <span className="icon-[tabler--lock] absolute left-3 top-1/2 size-4 -translate-y-1/2 text-gray-400" />
              <input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                autoComplete="current-password"
                className="w-full rounded-xl border border-gray-200 py-2.5 pl-9 pr-3 text-sm focus:border-brand-500 focus:outline-none"
                placeholder="••••••••"
              />
            </div>
          </div>

          {error && (
            <div className="flex items-center gap-2 rounded-xl bg-red-50 px-3 py-2.5 text-sm text-red-600">
              <span className="icon-[tabler--alert-circle] size-4" />
              {error}
            </div>
          )}

          <button
            type="submit"
            disabled={loading}
            className="flex w-full items-center justify-center gap-2 rounded-xl bg-brand-500 py-2.5 text-sm font-semibold text-white hover:bg-brand-600 disabled:opacity-50"
          >
            {loading ? (
              <span className="icon-[svg-spinners--3-dots-move] size-5" />
            ) : (
              <>
                <span className="icon-[tabler--login-2] size-4" />
                Sign In
              </>
            )}
          </button>
        </form>
      </div>
    </div>
  )
}
