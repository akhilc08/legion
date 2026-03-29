import { cn } from '@/lib/utils'

interface AgentStatusCardProps {
  label: string
  count: number
  colour: string // tailwind text colour class
}

export function AgentStatusCard({ label, count, colour }: AgentStatusCardProps) {
  return (
    <div className="rounded-lg border border-zinc-800 bg-zinc-900 px-4 py-3">
      <p className={cn('text-2xl font-semibold tabular-nums', colour)}>{count}</p>
      <p className="text-xs text-zinc-500 mt-0.5">{label}</p>
    </div>
  )
}
