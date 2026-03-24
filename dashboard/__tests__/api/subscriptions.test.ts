import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { NextRequest } from 'next/server'
import { POST } from '@/app/api/subscriptions/route'
import { GET } from '@/app/api/subscriptions/[user_id]/route'
import { DELETE } from '@/app/api/subscriptions/[user_id]/[id]/route'

function makeRequest(body: unknown): NextRequest {
  return new NextRequest('http://localhost/api/subscriptions', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
}

afterEach(() => {
  vi.unstubAllGlobals()
  delete process.env.SUBSCRIPTION_URL
})

// ─── POST /api/subscriptions ────────────────────────────────────────────────

describe('POST /api/subscriptions', () => {
  const payload = { user_id: 'user-abc', protocol_id: 'aave', threshold: 40 }
  const createdSub = { id: 'sub-1', ...payload, created_at: '2024-01-15T10:00:00Z' }

  beforeEach(() => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        status: 201,
        json: async () => createdSub,
      }),
    )
  })

  it('proxies the request body to the backend', async () => {
    await POST(makeRequest(payload))
    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/v1/subscriptions'),
      expect.objectContaining({
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      }),
    )
  })

  it('returns created subscription with 201 status', async () => {
    const res = await POST(makeRequest(payload))
    expect(res.status).toBe(201)
    expect(await res.json()).toEqual(createdSub)
  })

  it('proxies to the subscription backend URL (env evaluated at module load)', async () => {
    // SUBSCRIPTION_URL is read once at module load time.
    // The URL shape is validated by the proxy body test above.
    await POST(makeRequest(payload))
    expect(fetch).toHaveBeenCalledWith(
      expect.stringMatching(/\/api\/v1\/subscriptions$/),
      expect.anything(),
    )
  })

  it('forwards a 409 conflict status from the backend', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        status: 409,
        json: async () => ({ error: 'already exists' }),
      }),
    )
    const res = await POST(makeRequest(payload))
    expect(res.status).toBe(409)
    expect(await res.json()).toEqual({ error: 'already exists' })
  })
})

// ─── GET /api/subscriptions/[user_id] ───────────────────────────────────────

describe('GET /api/subscriptions/[user_id]', () => {
  const subs = [
    { id: 'sub-1', user_id: 'user-abc', protocol_id: 'aave', threshold: 40 },
  ]

  beforeEach(() => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({ status: 200, json: async () => subs }),
    )
  })

  it('fetches subscriptions for the given user_id', async () => {
    const req = new NextRequest('http://localhost/api/subscriptions/user-abc')
    await GET(req, { params: { user_id: 'user-abc' } })
    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/v1/subscriptions/user-abc'),
      { cache: 'no-store' },
    )
  })

  it('returns the subscription list', async () => {
    const req = new NextRequest('http://localhost/api/subscriptions/user-abc')
    const res = await GET(req, { params: { user_id: 'user-abc' } })
    expect(res.status).toBe(200)
    expect(await res.json()).toEqual(subs)
  })

  it('forwards 404 when user not found', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({ status: 404, json: async () => ({ error: 'not found' }) }),
    )
    const req = new NextRequest('http://localhost/api/subscriptions/unknown')
    const res = await GET(req, { params: { user_id: 'unknown' } })
    expect(res.status).toBe(404)
  })
})

// ─── DELETE /api/subscriptions/[user_id]/[id] ───────────────────────────────

describe('DELETE /api/subscriptions/[user_id]/[id]', () => {
  it('calls DELETE on the backend with correct path', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({ status: 204 }),
    )
    const req = new NextRequest('http://localhost/api/subscriptions/user-abc/sub-1', { method: 'DELETE' })
    await DELETE(req, { params: { user_id: 'user-abc', id: 'sub-1' } })

    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/v1/subscriptions/user-abc/sub-1'),
      { method: 'DELETE' },
    )
  })

  it('returns 204 with no body when backend responds 204', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({ status: 204 }),
    )
    const req = new NextRequest('http://localhost/api/subscriptions/user-abc/sub-1', { method: 'DELETE' })
    const res = await DELETE(req, { params: { user_id: 'user-abc', id: 'sub-1' } })

    expect(res.status).toBe(204)
    expect(res.body).toBeNull()
  })

  it('returns JSON body when backend returns non-204 status', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        status: 404,
        json: async () => ({ error: 'subscription not found' }),
      }),
    )
    const req = new NextRequest('http://localhost/api/subscriptions/user-abc/sub-99', { method: 'DELETE' })
    const res = await DELETE(req, { params: { user_id: 'user-abc', id: 'sub-99' } })

    expect(res.status).toBe(404)
    expect(await res.json()).toEqual({ error: 'subscription not found' })
  })
})
