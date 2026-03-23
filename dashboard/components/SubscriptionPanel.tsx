'use client'

import { useState } from 'react'
import { Protocol, Subscription } from '@/lib/types'

interface Props {
  userId: string
  protocols: Protocol[]
  subscriptions: Subscription[]
  onRefresh: () => void
}

export function SubscriptionPanel({ userId, protocols, subscriptions, onRefresh }: Props) {
  const [protocolId, setProtocolId] = useState('')
  const [threshold, setThreshold] = useState(50)
  const [creating, setCreating] = useState(false)
  const [error, setError] = useState('')

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault()
    if (!userId || !protocolId) return
    setCreating(true)
    setError('')
    try {
      const res = await fetch('/api/subscriptions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ user_id: userId, protocol_id: protocolId, threshold }),
      })
      if (!res.ok) {
        const data = await res.json()
        setError(data.error ?? 'Failed to create subscription')
        return
      }
      await onRefresh()
      setProtocolId('')
      setThreshold(50)
    } finally {
      setCreating(false)
    }
  }

  async function handleDelete(subId: string) {
    const res = await fetch(`/api/subscriptions/${userId}/${subId}`, {
      method: 'DELETE',
    })
    if (res.ok || res.status === 204) {
      await onRefresh()
    }
  }

  return (
    <div className="space-y-3">
      <form
        onSubmit={handleCreate}
        className="bg-gray-900 rounded-lg border border-gray-800 p-4 space-y-3"
      >
        <p className="text-sm font-medium text-gray-300">New subscription</p>
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="text-xs text-gray-500 mb-1 block">Protocol</label>
            <select
              value={protocolId}
              onChange={e => setProtocolId(e.target.value)}
              required
              className="w-full bg-gray-800 border border-gray-700 rounded px-3 py-2 text-sm text-gray-100 focus:outline-none focus:border-blue-500"
            >
              <option value="">Select...</option>
              {protocols.map(p => (
                <option key={p.id} value={p.id}>{p.name}</option>
              ))}
            </select>
          </div>
          <div>
            <label className="text-xs text-gray-500 mb-1 block">Alert below score</label>
            <input
              type="number"
              value={threshold}
              onChange={e => setThreshold(Number(e.target.value))}
              min={1}
              max={100}
              required
              className="w-full bg-gray-800 border border-gray-700 rounded px-3 py-2 text-sm text-gray-100 focus:outline-none focus:border-blue-500"
            />
          </div>
        </div>
        {error && <p className="text-xs text-red-400">{error}</p>}
        <button
          type="submit"
          disabled={creating || !userId}
          className="w-full bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white text-sm font-medium py-2 px-4 rounded transition-colors"
        >
          {creating ? 'Creating...' : 'Subscribe'}
        </button>
      </form>

      <div className="space-y-2">
        {subscriptions.length === 0 && (
          <p className="text-sm text-gray-500 text-center py-4">No subscriptions yet.</p>
        )}
        {subscriptions.map(sub => (
          <div
            key={sub.id}
            className="bg-gray-900 rounded-lg border border-gray-800 px-4 py-3 flex items-center justify-between"
          >
            <div>
              <span className="text-sm font-medium text-gray-200 capitalize">
                {sub.protocol_id}
              </span>
              <span className="text-xs text-gray-500 ml-2">
                alert below {sub.threshold}
              </span>
            </div>
            <button
              onClick={() => handleDelete(sub.id)}
              className="text-xs text-red-400 hover:text-red-300 transition-colors"
            >
              Remove
            </button>
          </div>
        ))}
      </div>
    </div>
  )
}
