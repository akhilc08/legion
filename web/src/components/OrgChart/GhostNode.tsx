import { memo } from 'react'
import { Handle, Position, type NodeProps, type Node } from '@xyflow/react'
import type { PendingHire } from '@/lib/types'

export type GhostNodeType = Node<{
  hire: PendingHire
  onApprove: () => void
  onReject: () => void
}, 'ghostNode'>

export const GhostNode = memo(function GhostNode({ data }: NodeProps<GhostNodeType>) {
  const { hire, onApprove, onReject } = data
  return (
    <>
      <Handle type="target" position={Position.Top} className="!bg-zinc-600" />
      <div className="rounded-md border border-dashed border-zinc-600 bg-zinc-900/60 px-3 py-2 min-w-[140px] select-none opacity-70">
        <p className="text-[10px] text-zinc-500 mb-0.5">Awaiting Approval</p>
        <p className="text-xs font-medium text-zinc-300 truncate max-w-[120px]">{hire.role_title}</p>
        <div className="flex gap-1 mt-2">
          <button
            onClick={(e) => { e.stopPropagation(); onApprove() }}
            className="flex-1 rounded text-[10px] py-0.5 bg-emerald-900/60 text-emerald-400 hover:bg-emerald-800/60 transition-colors"
          >
            Approve
          </button>
          <button
            onClick={(e) => { e.stopPropagation(); onReject() }}
            className="flex-1 rounded text-[10px] py-0.5 bg-red-900/60 text-red-400 hover:bg-red-800/60 transition-colors"
          >
            Reject
          </button>
        </div>
      </div>
      <Handle type="source" position={Position.Bottom} className="!bg-zinc-600" />
    </>
  )
})
