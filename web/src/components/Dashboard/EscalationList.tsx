import type { Notification } from '@/lib/types'

interface EscalationListProps {
  notifications: Notification[]
  onDismiss: (id: string) => void
}

export function EscalationList({ notifications, onDismiss }: EscalationListProps) {
  if (notifications.length === 0) {
    return <p className="text-sm text-zinc-600">No active notifications.</p>
  }

  return (
    <ul className="space-y-2">
      {notifications.map((n) => (
        <li
          key={n.id}
          className="flex items-start justify-between rounded-md border border-zinc-800 bg-zinc-900 px-3 py-2"
        >
          <div>
            <p className="text-xs font-medium text-zinc-300">{n.type}</p>
            <p className="text-xs text-zinc-500 mt-0.5">
              {new Date(n.created_at).toLocaleString()}
            </p>
          </div>
          <button
            className="text-zinc-600 hover:text-zinc-300 text-xs ml-4 shrink-0"
            onClick={() => onDismiss(n.id)}
          >
            dismiss
          </button>
        </li>
      ))}
    </ul>
  )
}
