import React from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { useAppStore } from '@/store/useAppStore'
import { Login } from '@/pages/Login'
import { Register } from '@/pages/Register'
import { CompanyList } from '@/pages/CompanyList'
import { CompanyShell } from '@/pages/CompanyShell'

class ErrorBoundary extends React.Component<
  { children: React.ReactNode },
  { error: Error | null }
> {
  state = { error: null }
  static getDerivedStateFromError(error: Error) { return { error } }
  render() {
    if (this.state.error) {
      return (
        <div className="flex min-h-screen items-center justify-center bg-zinc-950 p-8">
          <div className="max-w-lg space-y-3">
            <p className="text-red-400 font-mono text-sm font-semibold">Runtime error</p>
            <pre className="text-xs text-zinc-400 whitespace-pre-wrap font-mono bg-zinc-900 p-4 rounded-md border border-zinc-800">
              {(this.state.error as Error).message}
              {'\n\n'}
              {(this.state.error as Error).stack}
            </pre>
          </div>
        </div>
      )
    }
    return this.props.children
  }
}

function RequireAuth({ children }: { children: React.ReactNode }) {
  const token = useAppStore((s) => s.token)
  if (!token) return <Navigate to="/login" replace />
  return <>{children}</>
}

export function App() {
  return (
    <ErrorBoundary>
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
    </ErrorBoundary>
  )
}
