import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { ProtocolCard } from '@/components/ProtocolCard'
import type { Protocol } from '@/lib/types'

const base: Protocol = {
  id: 'aave',
  name: 'Aave',
  category: 'Lending',
  chain: 'Ethereum',
  health_score: 85,
  status: 'healthy',
  tvl_usd: 5_500_000_000,
  price_usd: 92.5,
  updated_at: '2024-01-15T10:30:00Z',
}

function makeProtocol(overrides: Partial<Protocol> = {}): Protocol {
  return { ...base, ...overrides }
}

describe('ProtocolCard', () => {
  it('renders the protocol name', () => {
    render(<ProtocolCard protocol={base} />)
    expect(screen.getByText('Aave')).toBeInTheDocument()
  })

  it('renders category and chain', () => {
    render(<ProtocolCard protocol={base} />)
    expect(screen.getByText('Lending · Ethereum')).toBeInTheDocument()
  })

  it('renders health score', () => {
    render(<ProtocolCard protocol={base} />)
    expect(screen.getByText('85')).toBeInTheDocument()
  })

  it('renders status badge', () => {
    render(<ProtocolCard protocol={base} />)
    expect(screen.getByText('healthy')).toBeInTheDocument()
  })

  it('renders TVL in billions with 2 decimals', () => {
    render(<ProtocolCard protocol={base} />)
    expect(screen.getByText('$5.50B')).toBeInTheDocument()
  })

  it('renders price with 2 decimals', () => {
    render(<ProtocolCard protocol={base} />)
    expect(screen.getByText('$92.50')).toBeInTheDocument()
  })

  it('health bar width matches health_score percentage', () => {
    const { container } = render(<ProtocolCard protocol={base} />)
    const bar = container.querySelector('[style]') as HTMLElement
    expect(bar.style.width).toBe('85%')
  })

  it('applies green card styling for healthy status', () => {
    const { container } = render(<ProtocolCard protocol={makeProtocol({ status: 'healthy' })} />)
    const card = container.firstChild as HTMLElement
    expect(card.className).toContain('border-green-800')
  })

  it('applies yellow card styling for degraded status', () => {
    const { container } = render(<ProtocolCard protocol={makeProtocol({ status: 'degraded', health_score: 55 })} />)
    const card = container.firstChild as HTMLElement
    expect(card.className).toContain('border-yellow-800')
  })

  it('applies red card styling for critical status', () => {
    const { container } = render(<ProtocolCard protocol={makeProtocol({ status: 'critical', health_score: 15 })} />)
    const card = container.firstChild as HTMLElement
    expect(card.className).toContain('border-red-800')
  })

  it('falls back to degraded styling for unknown status', () => {
    const protocol = makeProtocol({ status: 'unknown' as Protocol['status'] })
    const { container } = render(<ProtocolCard protocol={protocol} />)
    const card = container.firstChild as HTMLElement
    expect(card.className).toContain('border-yellow-800')
  })

  it('health bar width is 0% when health_score is 0', () => {
    const { container } = render(<ProtocolCard protocol={makeProtocol({ health_score: 0 })} />)
    const bar = container.querySelector('[style]') as HTMLElement
    expect(bar.style.width).toBe('0%')
  })

  it('health bar width is 100% when health_score is 100', () => {
    const { container } = render(<ProtocolCard protocol={makeProtocol({ health_score: 100 })} />)
    const bar = container.querySelector('[style]') as HTMLElement
    expect(bar.style.width).toBe('100%')
  })
})
