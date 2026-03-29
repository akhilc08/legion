import { memo } from 'react'
import { Handle, Position, type NodeProps, type Node } from '@xyflow/react'
import type { Agent } from '@/lib/types'
import { cn } from '@/lib/utils'

const STATUS_COLOURS: Record<Agent['status'], string> = {
  idle: 'bg-zinc-400', working: 'bg-emerald-400', paused: 'bg-blue-400',
  blocked: 'bg-amber-400', failed: 'bg-red-400', done: 'bg-zinc-600', degraded: 'bg-orange-400',
}

export type AgentNodeType = Node<{ agent: Agent }, 'agentNode'>

export const AgentNode = memo(function AgentNode({ data, selected }: NodeProps<AgentNodeType>) {
  const { agent } = data
  return (
    <>
      <Handle type="target" position={Position.Top} className="!bg-zinc-600" />
      <div className={cn('rounded-md border bg-zinc-900 px-3 py-2 min-w-[120px] cursor-pointer select-none',
        selected ? 'border-blue-500 shadow-[0_0_0_2px_rgba(59,130,246,0.3)]' : 'border-zinc-700')}>
        <div className="flex items-center gap-2">
          <span data-status-dot className={cn('h-2 w-2 rounded-full shrink-0', STATUS_COLOURS[agent.status])} />
          <span className="text-xs font-medium text-zinc-100 truncate max-w-[100px]">{agent.title}</span>
        </div>
        <p className="text-[10px] text-zinc-500 mt-0.5 pl-4">{agent.status}</p>
      </div>
      <Handle type="source" position={Position.Bottom} className="!bg-zinc-600" />
    </>
  )
})
