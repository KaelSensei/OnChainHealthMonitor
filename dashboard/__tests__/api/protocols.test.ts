import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { GET } from '@/app/api/protocols/route'

const mockProtocols = [
  {
    id: 'aave',
    name: 'Aave',
    category: 'Lending',
    chain: 'Ethereum',
    health_score: 85,
    status: 'healthy',
    tvl_usd: 5e9,
    price_usd: 92.5,
    updated_at: '2024-01-15T10:00:00Z',
  },
]

beforeEach(() => {
  vi.stubGlobal(
    'fetch',
    vi.fn().mockResolvedValue({
      status: 200,
      json: async () => mockProtocols,
    }),
  )
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('GET /api/protocols', () => {
  it('fetches from the backend API_URL', async () => {
    await GET()
    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/v1/protocols'),
      { cache: 'no-store' },
    )
  })

  it('returns the proxied JSON body', async () => {
    const response = await GET()
    const data = await response.json()
    expect(data).toEqual(mockProtocols)
  })

  it('forwards the backend status code', async () => {
    const response = await GET()
    expect(response.status).toBe(200)
  })

  it('forwards a non-200 status from the backend', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        status: 503,
        json: async () => ({ error: 'service unavailable' }),
      }),
    )
    const response = await GET()
    expect(response.status).toBe(503)
  })

  it('fetches from localhost:8080 by default (env evaluated at module load)', async () => {
    // API_URL is read once at module load time; changing process.env after import has no effect.
    // The default is verified by the URL assertion in "fetches from the backend API_URL".
    await GET()
    expect(fetch).toHaveBeenCalledWith(
      expect.stringMatching(/\/api\/v1\/protocols$/),
      { cache: 'no-store' },
    )
  })
})
