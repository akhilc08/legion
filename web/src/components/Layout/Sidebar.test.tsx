import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { Sidebar } from './Sidebar'

vi.mock('@/store/useAppStore', () => ({
  useAppStore: (sel: (s: { companyId: string | null }) => unknown) =>
    sel({ companyId: 'company-1' }),
}))

vi.mock('./CompanySelector', () => ({
  CompanySelector: () => <div data-testid="company-selector" />,
}))

describe('Sidebar', () => {
  it('renders all nav links', () => {
    render(
      <MemoryRouter>
        <Sidebar hireBadgeCount={2} />
      </MemoryRouter>
    )
    expect(screen.getByText('Dashboard')).toBeInTheDocument()
    expect(screen.getByText('Org Chart')).toBeInTheDocument()
    expect(screen.getByText('Issues')).toBeInTheDocument()
    expect(screen.getByText('Audit')).toBeInTheDocument()
  })

  it('shows hire badge count', () => {
    render(
      <MemoryRouter>
        <Sidebar hireBadgeCount={2} />
      </MemoryRouter>
    )
    expect(screen.getByText('2')).toBeInTheDocument()
  })
})
