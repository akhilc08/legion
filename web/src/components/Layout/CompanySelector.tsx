import { useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { apiClient } from '@/lib/api'
import type { Company } from '@/lib/types'
import { useAppStore } from '@/store/useAppStore'
import { cn } from '@/lib/utils'
import { ChevronsUpDown } from 'lucide-react'

export function CompanySelector() {
  const navigate = useNavigate()
  const companyId = useAppStore((s) => s.companyId)

  const { data: companies = [] } = useQuery<Company[]>({
    queryKey: ['companies'],
    queryFn: () => apiClient.get('/api/companies').then((r) => r.data),
  })

  const current = companies.find((c) => c.id === companyId)

  return (
    <div className="px-2">
      <button
        className={cn(
          'flex w-full items-center gap-2 rounded-md px-2 py-2 text-sm',
          'bg-zinc-800 hover:bg-zinc-700 text-zinc-300 transition-colors'
        )}
        onClick={() => navigate('/')}
      >
        <span className="truncate flex-1 text-left">{current?.name ?? 'Select company'}</span>
        <ChevronsUpDown className="h-4 w-4 shrink-0 text-zinc-500" />
      </button>
    </div>
  )
}
