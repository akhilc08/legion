import { NavLink } from 'react-router-dom'
import { useAppStore } from '@/store/useAppStore'
import { CompanySelector } from './CompanySelector'
import { cn } from '@/lib/utils'
import {
  LayoutDashboard,
  Network,
  CircleDot,
  UserPlus,
  ScrollText,
  FolderOpen,
} from 'lucide-react'

interface SidebarProps {
  hireBadgeCount: number
}

const navItems = [
  { label: 'Dashboard', icon: LayoutDashboard, path: 'dashboard' },
  { label: 'Org Chart', icon: Network, path: 'org-chart' },
  { label: 'Issues', icon: CircleDot, path: 'issues' },
  { label: 'Hiring', icon: UserPlus, path: 'hiring' },
  { label: 'Audit', icon: ScrollText, path: 'audit' },
  { label: 'Files', icon: FolderOpen, path: 'files' },
]

export function Sidebar({ hireBadgeCount }: SidebarProps) {
  const companyId = useAppStore((s) => s.companyId)
  const base = companyId ? `/companies/${companyId}` : '#'

  return (
    <aside className="flex h-screen w-52 flex-col border-r border-zinc-800 bg-zinc-950">
      <div className="flex h-14 items-center px-4 border-b border-zinc-800">
        <span className="text-sm font-semibold text-zinc-100 tracking-wide">legion</span>
      </div>

      <nav className="flex-1 space-y-1 p-2 pt-3">
        {navItems.map(({ label, icon: Icon, path }) => (
          <NavLink
            key={path}
            to={`${base}/${path}`}
            className={({ isActive }) =>
              cn(
                'flex items-center gap-3 rounded-md px-3 py-2 text-sm transition-colors',
                isActive
                  ? 'bg-zinc-800 text-zinc-100'
                  : 'text-zinc-400 hover:bg-zinc-800/60 hover:text-zinc-200'
              )
            }
          >
            <Icon className="h-4 w-4 shrink-0" />
            <span className="flex-1">{label}</span>
            {label === 'Hiring' && hireBadgeCount > 0 && (
              <span className="rounded-full bg-red-600 px-1.5 py-0.5 text-xs font-medium text-white leading-none">
                {hireBadgeCount}
              </span>
            )}
          </NavLink>
        ))}
      </nav>

      <div className="border-t border-zinc-800 py-3">
        <CompanySelector />
      </div>
    </aside>
  )
}
