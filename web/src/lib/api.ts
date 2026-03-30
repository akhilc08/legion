import axios from 'axios'

export const apiClient = axios.create({
  baseURL: '/',
  headers: { 'Content-Type': 'application/json' },
})

apiClient.interceptors.request.use((config) => {
  const token = localStorage.getItem('legion_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

apiClient.interceptors.response.use(
  (res) => res,
  (err) => {
    const url: string = err.config?.url ?? ''
    const isAuthEndpoint = url.includes('/api/auth/')
    if (err.response?.status === 401 && !isAuthEndpoint) {
      localStorage.removeItem('legion_token')
      window.location.href = '/'
    }
    return Promise.reject(err)
  }
)
