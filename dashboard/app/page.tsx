import { Dashboard } from '@/components/Dashboard'
import { Protocol } from '@/lib/types'

export const dynamic = 'force-dynamic'

async function getProtocols(): Promise<Protocol[]> {
  try {
    const apiUrl = process.env.API_URL ?? 'http://localhost:8080'
    const res = await fetch(`${apiUrl}/api/v1/protocols`, { cache: 'no-store' })
    if (!res.ok) return []
    const data = await res.json()
    return data.protocols ?? []
  } catch {
    return []
  }
}

export default async function Home() {
  const protocols = await getProtocols()
  return <Dashboard initialProtocols={protocols} />
}
