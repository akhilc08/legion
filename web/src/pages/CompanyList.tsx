import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '@/lib/api'
import type { Company } from '@/lib/types'
import { useAppStore } from '@/store/useAppStore'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

export function CompanyList() {
  const navigate = useNavigate()
  const qc = useQueryClient()
  const setToken = useAppStore((s) => s.setToken)
  const [name, setName] = useState('')
  const [goal, setGoal] = useState('')

  const { data: companies = [], isLoading } = useQuery<Company[]>({
    queryKey: ['companies'],
    queryFn: () => apiClient.get('/api/companies').then((r) => r.data),
  })

  const create = useMutation({
    mutationFn: (body: { name: string; goal: string }) =>
      apiClient.post('/api/companies', body).then((r) => r.data as Company),
    onSuccess: (company) => {
      qc.invalidateQueries({ queryKey: ['companies'] })
      navigate(`/companies/${company.id}/dashboard`)
    },
  })

  return (
    <div className="flex min-h-screen flex-col items-center justify-center bg-zinc-950 p-6 gap-6">
      <div className="w-full max-w-md">
        <div className="flex items-center justify-between mb-6">
          <h1 className="text-xl font-semibold text-zinc-100">Your companies</h1>
          <Button
            variant="ghost"
            size="sm"
            className="text-zinc-500"
            onClick={() => setToken(null)}
          >
            Sign out
          </Button>
        </div>

        {isLoading ? (
          <p className="text-zinc-500 text-sm">Loading…</p>
        ) : (
          <div className="space-y-2 mb-6">
            {companies.map((c) => (
              <button
                key={c.id}
                className="w-full text-left rounded-md border border-zinc-800 bg-zinc-900 px-4 py-3 hover:bg-zinc-800 transition-colors"
                onClick={() => navigate(`/companies/${c.id}/dashboard`)}
              >
                <p className="text-sm font-medium text-zinc-100">{c.name}</p>
                <p className="text-xs text-zinc-500 mt-0.5 line-clamp-1">{c.goal}</p>
              </button>
            ))}
          </div>
        )}

        <Card className="border-zinc-800 bg-zinc-900">
          <CardHeader className="pb-3">
            <CardTitle className="text-sm text-zinc-300">New company</CardTitle>
          </CardHeader>
          <CardContent>
            <form
              className="space-y-3"
              onSubmit={(e) => {
                e.preventDefault()
                create.mutate({ name, goal })
              }}
            >
              <Input
                placeholder="Name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                required
                className="bg-zinc-800 border-zinc-700 text-zinc-100"
              />
              <Input
                placeholder="Goal"
                value={goal}
                onChange={(e) => setGoal(e.target.value)}
                className="bg-zinc-800 border-zinc-700 text-zinc-100"
              />
              <Button type="submit" disabled={create.isPending} className="w-full">
                {create.isPending ? 'Creating…' : 'Create'}
              </Button>
            </form>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
