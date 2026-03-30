import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { useAutoLogin } from '@/hooks/useAutoLogin'
import { CompanyList } from '@/pages/CompanyList'
import { CompanyShell } from '@/pages/CompanyShell'
import { useAppStore } from '@/store/useAppStore'

function AppRoutes() {
  useAutoLogin()
  const token = useAppStore((s) => s.token)

  if (!token) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-zinc-950">
        <p className="text-sm text-zinc-500">Starting…</p>
      </div>
    )
  }

  return (
    <Routes>
      <Route path="/" element={<CompanyList />} />
      <Route path="/companies/:companyId/*" element={<CompanyShell />} />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}

export function App() {
  return (
    <BrowserRouter>
      <AppRoutes />
    </BrowserRouter>
  )
}
