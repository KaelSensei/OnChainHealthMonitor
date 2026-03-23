import { Protocol } from '@/lib/types'

const statusConfig = {
  healthy: {
    card: 'border-green-800 bg-green-950/40',
    badge: 'bg-green-900 text-green-300',
    bar: 'bg-green-500',
  },
  degraded: {
    card: 'border-yellow-800 bg-yellow-950/40',
    badge: 'bg-yellow-900 text-yellow-300',
    bar: 'bg-yellow-500',
  },
  critical: {
    card: 'border-red-800 bg-red-950/40',
    badge: 'bg-red-900 text-red-300',
    bar: 'bg-red-500',
  },
}

interface Props {
  protocol: Protocol
}

export function ProtocolCard({ protocol }: Props) {
  const cfg = statusConfig[protocol.status] ?? statusConfig.degraded

  return (
    <div className={`rounded-lg border p-4 ${cfg.card}`}>
      <div className="flex items-start justify-between mb-3">
        <div>
          <h3 className="font-semibold text-white">{protocol.name}</h3>
          <p className="text-xs text-gray-400 mt-0.5">
            {protocol.category} · {protocol.chain}
          </p>
        </div>
        <div className="text-right">
          <span className="text-2xl font-bold text-white">{protocol.health_score}</span>
          <span className={`block text-xs px-2 py-0.5 rounded-full mt-1 ${cfg.badge}`}>
            {protocol.status}
          </span>
        </div>
      </div>

      <div className="mb-3">
        <div className="h-1.5 bg-gray-800 rounded-full overflow-hidden">
          <div
            className={`h-full rounded-full transition-all duration-700 ${cfg.bar}`}
            style={{ width: `${protocol.health_score}%` }}
          />
        </div>
      </div>

      <div className="grid grid-cols-2 gap-2 text-xs">
        <div>
          <p className="text-gray-500">TVL</p>
          <p className="text-gray-200 font-medium">
            ${(protocol.tvl_usd / 1e9).toFixed(2)}B
          </p>
        </div>
        <div>
          <p className="text-gray-500">Price</p>
          <p className="text-gray-200 font-medium">${protocol.price_usd.toFixed(2)}</p>
        </div>
      </div>

      <p className="text-xs text-gray-600 mt-2">
        {new Date(protocol.updated_at).toLocaleTimeString()}
      </p>
    </div>
  )
}
