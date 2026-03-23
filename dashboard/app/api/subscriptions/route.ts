import { NextRequest, NextResponse } from 'next/server'

const SUBSCRIPTION_URL = process.env.SUBSCRIPTION_URL ?? 'http://localhost:8084'

export async function POST(req: NextRequest) {
  const body = await req.json()
  const res = await fetch(`${SUBSCRIPTION_URL}/api/v1/subscriptions`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  const data = await res.json()
  return NextResponse.json(data, { status: res.status })
}
