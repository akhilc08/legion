import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { useAppStore } from '@/store/useAppStore'
import { Login } from '@/pages/Login'
import { Register } from '@/pages/Register'
import { CompanyList } from '@/pages/CompanyList'
import { CompanyShell } from '@/pages/CompanyShell'

function RequireAuth({ children }: { children: React.ReactNode }) {
  const token = useAppStore((s) => s.token)
  if (!token) return <Navigate to="/login" replace />
  return <>{children}</>
}

export function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route path="/register" element={<Register />} />
        <Route
          path="/"
          element={
            <RequireAuth>
              <CompanyList />
            </RequireAuth>
          }
        />
        <Route
          path="/companies/:companyId/*"
          element={
            <RequireAuth>
              <CompanyShell />
            </RequireAuth>
          }
        />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  )
}
