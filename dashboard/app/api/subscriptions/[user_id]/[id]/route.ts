import { NextRequest, NextResponse } from 'next/server'

const SUBSCRIPTION_URL = process.env.SUBSCRIPTION_URL ?? 'http://localhost:8084'

export async function DELETE(
  _req: NextRequest,
  { params }: { params: { user_id: string; id: string } }
) {
  const res = await fetch(
    `${SUBSCRIPTION_URL}/api/v1/subscriptions/${params.user_id}/${params.id}`,
    { method: 'DELETE' }
  )
  if (res.status === 204) {
    return new NextResponse(null, { status: 204 })
  }
  const data = await res.json()
  return NextResponse.json(data, { status: res.status })
}
