import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { AlertFeed } from '@/components/AlertFeed'
import type { AlertMessage } from '@/lib/types'

function makeAlert(overrides: Partial<AlertMessage> = {}): AlertMessage {
  return {
    user_id: 'user-1',
    protocol_id: 'aave',
    score: 25,
    label: 'degraded',
    threshold: 30,
    message: 'Score dropped below threshold',
    fired_at: '2024-01-15T10:30:00Z',
    ...overrides,
  }
}

describe('AlertFeed', () => {
  describe('empty state', () => {
    it('shows empty state message when no alerts', () => {
      render(<AlertFeed alerts={[]} />)
      expect(
        screen.getByText(/No alerts yet/i),
      ).toBeInTheDocument()
    })

    it('does not render a list container when empty', () => {
      const { container } = render(<AlertFeed alerts={[]} />)
      expect(container.querySelector('.space-y-2')).not.toBeInTheDocument()
    })
  })

  describe('alert list', () => {
    it('renders all alerts', () => {
      const alerts = [
        makeAlert({ protocol_id: 'aave', score: 10 }),
        makeAlert({ protocol_id: 'compound', score: 25 }),
        makeAlert({ protocol_id: 'uniswap', score: 45 }),
      ]
      render(<AlertFeed alerts={alerts} />)
      expect(screen.getByText('aave')).toBeInTheDocument()
      expect(screen.getByText('compound')).toBeInTheDocument()
      expect(screen.getByText('uniswap')).toBeInTheDocument()
    })

    it('renders score for each alert', () => {
      render(<AlertFeed alerts={[makeAlert({ score: 18 })]} />)
      expect(screen.getByText('score: 18')).toBeInTheDocument()
    })

    it('renders the alert message', () => {
      render(<AlertFeed alerts={[makeAlert({ message: 'Critical drop detected' })]} />)
      expect(screen.getByText('Critical drop detected')).toBeInTheDocument()
    })

    it('applies red border for score < 20', () => {
      const { container } = render(<AlertFeed alerts={[makeAlert({ score: 15 })]} />)
      const item = container.querySelector('.border-l-4') as HTMLElement
      expect(item.className).toContain('border-red-500')
    })

    it('applies orange border for score between 20 and 29', () => {
      const { container } = render(<AlertFeed alerts={[makeAlert({ score: 25 })]} />)
      const item = container.querySelector('.border-l-4') as HTMLElement
      expect(item.className).toContain('border-orange-500')
    })

    it('applies yellow border for score >= 30', () => {
      const { container } = render(<AlertFeed alerts={[makeAlert({ score: 35 })]} />)
      const item = container.querySelector('.border-l-4') as HTMLElement
      expect(item.className).toContain('border-yellow-500')
    })

    it('applies yellow border when score equals exactly 30', () => {
      const { container } = render(<AlertFeed alerts={[makeAlert({ score: 30 })]} />)
      const item = container.querySelector('.border-l-4') as HTMLElement
      expect(item.className).toContain('border-yellow-500')
    })

    it('applies red border at score boundary 19', () => {
      const { container } = render(<AlertFeed alerts={[makeAlert({ score: 19 })]} />)
      const item = container.querySelector('.border-l-4') as HTMLElement
      expect(item.className).toContain('border-red-500')
    })

    it('renders multiple alerts in order', () => {
      const alerts = [
        makeAlert({ protocol_id: 'first', score: 10 }),
        makeAlert({ protocol_id: 'second', score: 20 }),
      ]
      render(<AlertFeed alerts={alerts} />)
      const items = screen.getAllByText(/first|second/)
      expect(items[0].textContent).toBe('first')
      expect(items[1].textContent).toBe('second')
    })
  })
})
