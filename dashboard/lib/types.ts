export interface Protocol {
  id: string
  name: string
  category: string
  chain: string
  health_score: number
  status: 'healthy' | 'degraded' | 'critical'
  tvl_usd: number
  price_usd: number
  updated_at: string
}

export interface Subscription {
  id: string
  user_id: string
  protocol_id: string
  threshold: number
  created_at: string
}

export interface AlertMessage {
  user_id: string
  protocol_id: string
  score: number
  label: string
  threshold: number
  message: string
  fired_at: string
}
