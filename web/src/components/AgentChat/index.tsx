interface AgentChatProps {
  agentId: string
  companyId: string
}

export function AgentChat({ agentId, companyId }: AgentChatProps) {
  void agentId; void companyId
  return (
    <div className="flex items-center justify-center h-32 text-xs text-zinc-600">
      Chat coming in Phase 5
    </div>
  )
}
