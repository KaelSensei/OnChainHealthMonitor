'use client'

import { useState, useEffect, useCallback } from 'react'
import { Protocol, Subscription, AlertMessage } from '@/lib/types'
import { ProtocolCard } from './ProtocolCard'
import { SubscriptionPanel } from './SubscriptionPanel'
import { AlertFeed } from './AlertFeed'

export function Dashboard({ initialProtocols }: { initialProtocols: Protocol[] }) {
  const [userId, setUserId] = useState('')
  const [protocols, setProtocols] = useState<Protocol[]>(initialProtocols)
  const [subscriptions, setSubscriptions] = useState<Subscription[]>([])
  const [alerts, setAlerts] = useState<AlertMessage[]>([])

  // Initialise user ID from localStorage (generated once, persisted across sessions)
  useEffect(() => {
    let id = localStorage.getItem('onchain_user_id')
    if (!id) {
      id = crypto.randomUUID()
      localStorage.setItem('onchain_user_id', id)
    }
    setUserId(id)
  }, [])

  // Poll protocol health every 5 s
  useEffect(() => {
    const interval = setInterval(async () => {
      try {
        const res = await fetch('/api/protocols')
        if (res.ok) {
          const data = await res.json()
          setProtocols(data.protocols ?? [])
        }
      } catch {
        // silent - keep stale data visible
      }
    }, 5000)
    return () => clearInterval(interval)
  }, [])

  // Fetch subscriptions whenever userId is available
  const fetchSubscriptions = useCallback(async () => {
    if (!userId) return
    try {
      const res = await fetch(`/api/subscriptions/${userId}`)
      if (res.ok) {
        const data = await res.json()
        setSubscriptions(data.subscriptions ?? [])
      }
    } catch {
      // silent
    }
  }, [userId])

  useEffect(() => {
    fetchSubscriptions()
  }, [fetchSubscriptions])

  // WebSocket - browser connects directly to subscription service port
  useEffect(() => {
    if (!userId) return
    const ws = new WebSocket(`ws://${window.location.hostname}:8084/ws?user_id=${userId}`)

    ws.onmessage = event => {
      try {
        const alert: AlertMessage = JSON.parse(event.data as string)
        setAlerts(prev => [alert, ...prev].slice(0, 50))
      } catch {
        // ignore malformed frames
      }
    }

    ws.onerror = () => console.warn('[ws] connection error - alerts unavailable')

    return () => ws.close()
  }, [userId])

  return (
    <div className="min-h-screen text-gray-100">
      <header className="border-b border-gray-800 px-6 py-4">
        <div className="max-w-7xl mx-auto flex items-start justify-between">
          <div>
            <h1 className="text-lg font-bold text-white tracking-tight">
              OnChain Health Monitor
            </h1>
            <p className="text-xs text-gray-500 mt-0.5">Real-time DeFi protocol health</p>
          </div>
          <div className="text-right">
            <p className="text-xs text-gray-600">user id</p>
            <p
              className="text-xs font-mono text-gray-400 max-w-[220px] truncate cursor-pointer"
              title={userId}
              onClick={() => userId && navigator.clipboard.writeText(userId)}
            >
              {userId || 'loading...'}
            </p>
          </div>
        </div>
      </header>

      <main className="max-w-7xl mx-auto px-6 py-8 space-y-10">
        <section>
          <h2 className="text-xs font-semibold text-gray-500 uppercase tracking-wider mb-4">
            Protocol Health
          </h2>
          {protocols.length === 0 ? (
            <p className="text-sm text-gray-500">
              Waiting for data from the API service...
            </p>
          ) : (
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
              {protocols.map(p => (
                <ProtocolCard key={p.id} protocol={p} />
              ))}
            </div>
          )}
        </section>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-10">
          <section>
            <h2 className="text-xs font-semibold text-gray-500 uppercase tracking-wider mb-4">
              Subscriptions
            </h2>
            <SubscriptionPanel
              userId={userId}
              protocols={protocols}
              subscriptions={subscriptions}
              onRefresh={fetchSubscriptions}
            />
          </section>

          <section>
            <h2 className="text-xs font-semibold text-gray-500 uppercase tracking-wider mb-4">
              Alert Feed
              {alerts.length > 0 && (
                <span className="ml-2 text-xs font-normal text-gray-600 normal-case">
                  ({alerts.length})
                </span>
              )}
            </h2>
            <AlertFeed alerts={alerts} />
          </section>
        </div>
      </main>
    </div>
  )
}
