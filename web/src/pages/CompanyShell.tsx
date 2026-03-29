import { useEffect } from 'react'
import { useParams, Routes, Route, Navigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { useAppStore } from '@/store/useAppStore'
import { Sidebar } from '@/components/Layout/Sidebar'
import { apiClient } from '@/lib/api'
import type { PendingHire } from '@/lib/types'

// Stub components — will be replaced by real implementations in later tasks
function Dashboard() { return <div>Dashboard</div> }
function OrgChart() { return <div>OrgChart</div> }
function Issues() { return <div>Issues</div> }
function Hiring() { return <div>Hiring</div> }
function Audit() { return <div>Audit</div> }

export function CompanyShell() {
  const { companyId } = useParams<{ companyId: string }>()
  const setCompanyId = useAppStore((s) => s.setCompanyId)

  useEffect(() => {
    if (companyId) setCompanyId(companyId)
  }, [companyId, setCompanyId])

  const { data: hires = [] } = useQuery<PendingHire[]>({
    queryKey: ['hires', companyId],
    queryFn: () =>
      apiClient.get(`/api/companies/${companyId}/hires`).then((r) => r.data),
    enabled: !!companyId,
  })

  const pendingCount = hires.filter((h) => h.status === 'pending').length

  return (
    <div className="flex h-screen overflow-hidden">
      <Sidebar hireBadgeCount={pendingCount} />
      <main className="flex-1 overflow-auto bg-zinc-950">
        <Routes>
          <Route index element={<Navigate to="dashboard" replace />} />
          <Route path="dashboard" element={<Dashboard />} />
          <Route path="org-chart" element={<OrgChart />} />
          <Route path="issues" element={<Issues />} />
          <Route path="hiring" element={<Hiring />} />
          <Route path="audit" element={<Audit />} />
        </Routes>
      </main>
    </div>
  )
}
