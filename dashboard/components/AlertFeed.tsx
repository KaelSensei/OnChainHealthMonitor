import { AlertMessage } from '@/lib/types'

interface Props {
  alerts: AlertMessage[]
}

function alertStyle(score: number) {
  if (score < 20) return 'border-red-500 bg-red-500/10'
  if (score < 30) return 'border-orange-500 bg-orange-500/10'
  return 'border-yellow-500 bg-yellow-500/10'
}

export function AlertFeed({ alerts }: Props) {
  if (alerts.length === 0) {
    return (
      <div className="bg-gray-900 rounded-lg border border-gray-800 p-6 text-center">
        <p className="text-sm text-gray-500">
          No alerts yet. Alerts fire when a protocol score crosses your threshold.
        </p>
      </div>
    )
  }

  return (
    <div className="space-y-2 max-h-96 overflow-y-auto">
      {alerts.map((alert, i) => (
        <div
          key={i}
          className={`rounded-lg border-l-4 px-4 py-3 ${alertStyle(alert.score)}`}
        >
          <div className="flex items-start justify-between">
            <div>
              <span className="text-sm font-medium text-gray-200 capitalize">
                {alert.protocol_id}
              </span>
              <span className="text-xs text-gray-400 ml-2">score: {alert.score}</span>
            </div>
            <span className="text-xs text-gray-500 shrink-0 ml-2">
              {new Date(alert.fired_at).toLocaleTimeString()}
            </span>
          </div>
          <p className="text-xs text-gray-400 mt-1">{alert.message}</p>
        </div>
      ))}
    </div>
  )
}
