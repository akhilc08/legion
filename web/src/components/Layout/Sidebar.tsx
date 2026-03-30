import { useState } from 'react'
import { NavLink, useParams } from 'react-router-dom'
import { useAppStore } from '@/store/useAppStore'
import { CompanySelector } from './CompanySelector'
import type { Agent } from '@/lib/types'
import { cn } from '@/lib/utils'
import {
  LayoutDashboard,
  Network,
  CircleDot,
  UserPlus,
  ScrollText,
  FolderOpen,
  ChevronRight,
} from 'lucide-react'

const STATUS_DOT: Record<Agent['status'], string> = {
  idle: 'bg-zinc-500',
  working: 'bg-emerald-400',
  paused: 'bg-blue-400',
  blocked: 'bg-amber-400',
  failed: 'bg-red-500',
  done: 'bg-zinc-600',
  degraded: 'bg-orange-400',
}

const navItems = [
  { label: 'Dashboard', icon: LayoutDashboard, path: 'dashboard' },
  { label: 'Org Chart', icon: Network, path: 'org-chart' },
  { label: 'Issues', icon: CircleDot, path: 'issues' },
  { label: 'Hiring', icon: UserPlus, path: 'hiring' },
  { label: 'Audit', icon: ScrollText, path: 'audit' },
  { label: 'Files', icon: FolderOpen, path: 'files' },
]

interface AgentRowProps {
  agent: Agent
  all: Agent[]
  depth: number
  hireBadgeCount: number
}

function AgentRow({ agent, all, depth, hireBadgeCount }: AgentRowProps) {
  const setAgentId = useAppStore((s) => s.setAgentId)
  const selectedAgentId = useAppStore((s) => s.agentId)
  const children = all.filter((a) => a.manager_id === agent.id)
  const [open, setOpen] = useState(depth < 2)

  const isBoard = agent.role === 'board'

  return (
    <div>
      <div
        className={cn(
          'flex items-center gap-1.5 rounded-md px-2 py-1.5 cursor-pointer text-sm transition-colors group',
          selectedAgentId === agent.id
            ? 'bg-zinc-800 text-zinc-100'
            : 'text-zinc-400 hover:bg-zinc-800/50 hover:text-zinc-200',
        )}
        style={{ paddingLeft: `${8 + depth * 14}px` }}
        onClick={() => setAgentId(agent.id)}
      >
        {children.length > 0 ? (
          <button
            className="shrink-0 text-zinc-600 hover:text-zinc-300 -ml-0.5"
            onClick={(e) => { e.stopPropagation(); setOpen((v) => !v) }}
          >
            <ChevronRight className={cn('h-3 w-3 transition-transform', open && 'rotate-90')} />
          </button>
        ) : (
          <span className="w-3 shrink-0" />
        )}
        <span className={cn('h-2 w-2 rounded-full shrink-0', STATUS_DOT[agent.status])} />
        <span className="flex-1 truncate text-xs font-medium">{isBoard ? 'Board (you)' : agent.title}</span>
        {isBoard && (
          <span className="text-[10px] text-zinc-600 shrink-0">human</span>
        )}
        {!isBoard && agent.role && (
          <span className="text-[10px] text-zinc-600 shrink-0 hidden group-hover:inline">{agent.role}</span>
        )}
        {agent.role === 'hiring' && hireBadgeCount > 0 && (
          <span className="rounded-full bg-red-600 px-1 text-[10px] text-white">{hireBadgeCount}</span>
        )}
      </div>
      {open && children.map((child) => (
        <AgentRow key={child.id} agent={child} all={all} depth={depth + 1} hireBadgeCount={hireBadgeCount} />
      ))}
    </div>
  )
}

interface SidebarProps {
  hireBadgeCount: number
  agents: Agent[]
}

export function Sidebar({ hireBadgeCount, agents }: SidebarProps) {
  const { companyId } = useParams<{ companyId: string }>()
  const base = companyId ? `/companies/${companyId}` : '#'

  // Roots: agents with no manager, or whose manager isn't in the list
  const agentIds = new Set(agents.map((a) => a.id))
  const roots = agents
    .filter((a) => !a.manager_id || !agentIds.has(a.manager_id))
    .sort((a, b) => (a.role === 'board' ? -1 : b.role === 'board' ? 1 : 0))

  return (
    <aside className="flex h-screen w-52 flex-col border-r border-zinc-800 bg-zinc-950">
      <div className="flex h-14 items-center px-4 border-b border-zinc-800 shrink-0">
        <span className="text-sm font-semibold text-zinc-100 tracking-wide">legion</span>
      </div>

      <nav className="space-y-0.5 p-2 pt-3 shrink-0">
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

      {agents.length > 0 && (
        <>
          <div className="mx-3 my-2 border-t border-zinc-800" />
          <p className="px-3 pb-1 text-[10px] font-semibold uppercase tracking-widest text-zinc-600">Agents</p>
          <div className="flex-1 overflow-y-auto p-1 space-y-0.5">
            {roots.map((root) => (
              <AgentRow key={root.id} agent={root} all={agents} depth={0} hireBadgeCount={hireBadgeCount} />
            ))}
          </div>
        </>
      )}

      <div className="border-t border-zinc-800 py-3 shrink-0">
        <CompanySelector />
      </div>
    </aside>
  )
}
