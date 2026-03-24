import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { SubscriptionPanel } from '@/components/SubscriptionPanel'
import type { Protocol, Subscription } from '@/lib/types'

const protocols: Protocol[] = [
  {
    id: 'aave',
    name: 'Aave',
    category: 'Lending',
    chain: 'Ethereum',
    health_score: 85,
    status: 'healthy',
    tvl_usd: 5e9,
    price_usd: 92,
    updated_at: '2024-01-15T10:00:00Z',
  },
  {
    id: 'compound',
    name: 'Compound',
    category: 'Lending',
    chain: 'Ethereum',
    health_score: 70,
    status: 'healthy',
    tvl_usd: 3e9,
    price_usd: 55,
    updated_at: '2024-01-15T10:00:00Z',
  },
]

const subscriptions: Subscription[] = [
  {
    id: 'sub-1',
    user_id: 'user-abc',
    protocol_id: 'aave',
    threshold: 40,
    created_at: '2024-01-15T09:00:00Z',
  },
]

const defaultProps = {
  userId: 'user-abc',
  protocols,
  subscriptions: [],
  onRefresh: vi.fn(),
}

beforeEach(() => {
  vi.restoreAllMocks()
  defaultProps.onRefresh = vi.fn()
})

describe('SubscriptionPanel', () => {
  describe('rendering', () => {
    it('renders the form title', () => {
      render(<SubscriptionPanel {...defaultProps} />)
      expect(screen.getByText('New subscription')).toBeInTheDocument()
    })

    it('renders protocol options in the select', () => {
      render(<SubscriptionPanel {...defaultProps} />)
      expect(screen.getByRole('option', { name: 'Aave' })).toBeInTheDocument()
      expect(screen.getByRole('option', { name: 'Compound' })).toBeInTheDocument()
    })

    it('renders default threshold value of 50', () => {
      render(<SubscriptionPanel {...defaultProps} />)
      const input = screen.getByRole('spinbutton')
      expect(input).toHaveValue(50)
    })

    it('shows "No subscriptions yet" when subscriptions list is empty', () => {
      render(<SubscriptionPanel {...defaultProps} subscriptions={[]} />)
      expect(screen.getByText('No subscriptions yet.')).toBeInTheDocument()
    })

    it('renders existing subscriptions', () => {
      render(<SubscriptionPanel {...defaultProps} subscriptions={subscriptions} />)
      expect(screen.getByText('aave')).toBeInTheDocument()
      expect(screen.getByText(/alert below 40/i)).toBeInTheDocument()
    })

    it('renders Remove button for each subscription', () => {
      render(<SubscriptionPanel {...defaultProps} subscriptions={subscriptions} />)
      expect(screen.getByRole('button', { name: 'Remove' })).toBeInTheDocument()
    })
  })

  describe('form submission', () => {
    it('calls POST /api/subscriptions with correct payload', async () => {
      const user = userEvent.setup()
      const mockFetch = vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({}),
      })
      vi.stubGlobal('fetch', mockFetch)

      render(<SubscriptionPanel {...defaultProps} />)

      await user.selectOptions(screen.getByRole('combobox'), 'aave')
      await user.clear(screen.getByRole('spinbutton'))
      await user.type(screen.getByRole('spinbutton'), '30')
      await user.click(screen.getByRole('button', { name: /subscribe/i }))

      await waitFor(() => {
        expect(mockFetch).toHaveBeenCalledWith('/api/subscriptions', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ user_id: 'user-abc', protocol_id: 'aave', threshold: 30 }),
        })
      })
    })

    it('calls onRefresh after successful creation', async () => {
      const user = userEvent.setup()
      vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: async () => ({}) }))

      render(<SubscriptionPanel {...defaultProps} />)
      await user.selectOptions(screen.getByRole('combobox'), 'aave')
      await user.click(screen.getByRole('button', { name: /subscribe/i }))

      await waitFor(() => expect(defaultProps.onRefresh).toHaveBeenCalledOnce())
    })

    it('shows error message when creation fails', async () => {
      const user = userEvent.setup()
      vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
        ok: false,
        json: async () => ({ error: 'Protocol not found' }),
      }))

      render(<SubscriptionPanel {...defaultProps} />)
      await user.selectOptions(screen.getByRole('combobox'), 'aave')
      await user.click(screen.getByRole('button', { name: /subscribe/i }))

      await waitFor(() => expect(screen.getByText('Protocol not found')).toBeInTheDocument())
    })

    it('shows fallback error message when response has no error field', async () => {
      const user = userEvent.setup()
      vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
        ok: false,
        json: async () => ({}),
      }))

      render(<SubscriptionPanel {...defaultProps} />)
      await user.selectOptions(screen.getByRole('combobox'), 'aave')
      await user.click(screen.getByRole('button', { name: /subscribe/i }))

      await waitFor(() =>
        expect(screen.getByText('Failed to create subscription')).toBeInTheDocument(),
      )
    })

    it('shows "Creating..." during submission', async () => {
      const user = userEvent.setup()
      let resolve: (v: unknown) => void
      const pending = new Promise(r => { resolve = r })
      vi.stubGlobal('fetch', vi.fn().mockReturnValue(pending))

      render(<SubscriptionPanel {...defaultProps} />)
      await user.selectOptions(screen.getByRole('combobox'), 'aave')
      await user.click(screen.getByRole('button', { name: /subscribe/i }))

      expect(screen.getByRole('button', { name: /creating/i })).toBeInTheDocument()
      resolve!({ ok: true, json: async () => ({}) })
    })

    it('does not submit when no protocol is selected', async () => {
      const user = userEvent.setup()
      const mockFetch = vi.fn()
      vi.stubGlobal('fetch', mockFetch)

      render(<SubscriptionPanel {...defaultProps} />)
      // Do NOT select a protocol
      await user.click(screen.getByRole('button', { name: /subscribe/i }))

      expect(mockFetch).not.toHaveBeenCalled()
    })

    it('resets form fields after successful creation', async () => {
      const user = userEvent.setup()
      vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: async () => ({}) }))

      render(<SubscriptionPanel {...defaultProps} />)
      await user.selectOptions(screen.getByRole('combobox'), 'aave')
      await user.clear(screen.getByRole('spinbutton'))
      await user.type(screen.getByRole('spinbutton'), '30')
      await user.click(screen.getByRole('button', { name: /subscribe/i }))

      await waitFor(() => {
        expect(screen.getByRole('combobox')).toHaveValue('')
        expect(screen.getByRole('spinbutton')).toHaveValue(50)
      })
    })
  })

  describe('delete subscription', () => {
    it('calls DELETE endpoint when Remove is clicked', async () => {
      const user = userEvent.setup()
      const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 200, json: async () => ({}) })
      vi.stubGlobal('fetch', mockFetch)

      render(<SubscriptionPanel {...defaultProps} subscriptions={subscriptions} />)
      await user.click(screen.getByRole('button', { name: 'Remove' }))

      await waitFor(() => {
        expect(mockFetch).toHaveBeenCalledWith('/api/subscriptions/user-abc/sub-1', {
          method: 'DELETE',
        })
      })
    })

    it('calls onRefresh after successful delete', async () => {
      const user = userEvent.setup()
      vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, status: 200, json: async () => ({}) }))

      render(<SubscriptionPanel {...defaultProps} subscriptions={subscriptions} />)
      await user.click(screen.getByRole('button', { name: 'Remove' }))

      await waitFor(() => expect(defaultProps.onRefresh).toHaveBeenCalledOnce())
    })

    it('calls onRefresh on 204 response', async () => {
      const user = userEvent.setup()
      vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status: 204, json: async () => ({}) }))

      render(<SubscriptionPanel {...defaultProps} subscriptions={subscriptions} />)
      await user.click(screen.getByRole('button', { name: 'Remove' }))

      await waitFor(() => expect(defaultProps.onRefresh).toHaveBeenCalledOnce())
    })
  })
})
