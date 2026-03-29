import type { PendingHire } from '@/lib/types'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'

interface HireCardProps {
  hire: PendingHire
  onApprove: () => void
  onReject: () => void
  isApproving: boolean
  isRejecting: boolean
}

export function HireCard({ hire, onApprove, onReject, isApproving, isRejecting }: HireCardProps) {
  return (
    <Card className="border-zinc-800 bg-zinc-900">
      <CardContent className="pt-4 space-y-3">
        <div>
          <p className="text-sm font-medium text-zinc-100">{hire.role_title}</p>
          <p className="text-xs text-zinc-500 mt-0.5">{hire.runtime}</p>
        </div>
        {hire.initial_task && (
          <div className="rounded-md bg-zinc-800 px-3 py-2">
            <p className="text-xs text-zinc-500 mb-0.5">Initial task</p>
            <p className="text-xs text-zinc-300">{hire.initial_task}</p>
          </div>
        )}
        <div className="flex items-center justify-between text-xs text-zinc-500">
          <span>Budget: {hire.budget_allocation.toLocaleString()} tokens</span>
          <span>{new Date(hire.created_at).toLocaleDateString()}</span>
        </div>
        <div className="flex gap-2 pt-1">
          <Button size="sm" className="flex-1" disabled={isApproving} onClick={onApprove}>
            {isApproving ? 'Approving…' : 'Approve'}
          </Button>
          <Button size="sm" variant="secondary" className="flex-1" disabled={isRejecting} onClick={onReject}>
            {isRejecting ? 'Rejecting…' : 'Reject'}
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}
