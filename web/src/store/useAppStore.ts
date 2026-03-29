import { create } from 'zustand'

interface AppStore {
  token: string | null
  companyId: string | null
  agentId: string | null
  setToken: (token: string | null) => void
  setCompanyId: (id: string | null) => void
  setAgentId: (id: string | null) => void
  getInitialToken: () => string | null
}

export const useAppStore = create<AppStore>((set) => ({
  token: localStorage.getItem('legion_token'),
  companyId: null,
  agentId: null,

  setToken: (token) => {
    if (token) {
      localStorage.setItem('legion_token', token)
    } else {
      localStorage.removeItem('legion_token')
    }
    set({ token })
  },

  setCompanyId: (companyId) => set({ companyId }),

  setAgentId: (agentId) => set({ agentId }),

  getInitialToken: () => localStorage.getItem('legion_token'),
}))
