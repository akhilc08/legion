import { useEffect } from 'react'
import { apiClient } from '@/lib/api'
import { useAppStore } from '@/store/useAppStore'

const DEFAULT_EMAIL = 'admin@legion.local'
const DEFAULT_PASSWORD = 'legion'

export function useAutoLogin() {
  const token = useAppStore((s) => s.token)
  const setToken = useAppStore((s) => s.setToken)

  useEffect(() => {
    if (token) return

    async function login() {
      try {
        const { data } = await apiClient.post('/api/auth/login', {
          email: DEFAULT_EMAIL,
          password: DEFAULT_PASSWORD,
        })
        setToken(data.token)
      } catch {
        // Account doesn't exist yet — register then login.
        try {
          await apiClient.post('/api/auth/register', {
            email: DEFAULT_EMAIL,
            password: DEFAULT_PASSWORD,
          })
          const { data } = await apiClient.post('/api/auth/login', {
            email: DEFAULT_EMAIL,
            password: DEFAULT_PASSWORD,
          })
          setToken(data.token)
        } catch (err) {
          console.error('auto-login failed', err)
        }
      }
    }

    login()
  }, [token, setToken])
}
