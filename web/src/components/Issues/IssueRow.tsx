import { useState } from 'react'
import type { Agent, Issue } from '@/lib/types'
import { cn } from '@/lib/utils'
import { ChevronDown, ChevronRight } from 'lucide-react'

const STATUS_COLOURS: Record<Issue['status'], string> = {
  pending: 'text-zinc-400', in_progress: 'text-blue-400',
  blocked: 'text-amber-400', done: 'text-emerald-400', failed: 'text-red-400',
}

interface IssueRowProps { issue: Issue; agents: Agent[] }

export function IssueRow({ issue, agents }: IssueRowProps) {
  const [open, setOpen] = useState(false)
  const assignee = agents.find((a) => a.id === issue.assignee_id)
  return (
    <>
      <tr className="border-b border-zinc-800 hover:bg-zinc-900/50 cursor-pointer" onClick={() => setOpen((o) => !o)}>
        <td className="px-3 py-2.5 text-sm text-zinc-200">{issue.title}</td>
        <td className="px-3 py-2.5 text-xs"><span className={STATUS_COLOURS[issue.status]}>{issue.status}</span></td>
        <td className="px-3 py-2.5 text-xs text-zinc-500">{assignee?.title ?? '—'}</td>
        <td className="px-3 py-2.5 text-xs text-zinc-600">{issue.attempt_count}</td>
        <td className="px-3 py-2.5 text-zinc-600 w-6">{open ? <ChevronDown className="h-3 w-3" /> : <ChevronRight className="h-3 w-3" />}</td>
      </tr>
      {open && (
        <tr className="border-b border-zinc-800 bg-zinc-900/30">
          <td colSpan={5} className="px-4 py-3 text-xs text-zinc-400 space-y-1">
            {issue.description && <p>{issue.description}</p>}
            {issue.last_failure_reason && <p className="text-red-400">Failure: {issue.last_failure_reason}</p>}
            {issue.output_path && <p className="text-zinc-500">Output: {issue.output_path}</p>}
          </td>
        </tr>
      )}
    </>
  )
}
