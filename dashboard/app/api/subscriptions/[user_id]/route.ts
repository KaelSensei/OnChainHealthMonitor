import { NextRequest, NextResponse } from 'next/server'

const SUBSCRIPTION_URL = process.env.SUBSCRIPTION_URL ?? 'http://localhost:8084'

export async function GET(
  _req: NextRequest,
  { params }: { params: { user_id: string } }
) {
  const res = await fetch(
    `${SUBSCRIPTION_URL}/api/v1/subscriptions/${params.user_id}`,
    { cache: 'no-store' }
  )
  const data = await res.json()
  return NextResponse.json(data, { status: res.status })
}
