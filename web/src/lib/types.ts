export type AgentRuntime = 'claude_code' | 'openclaw'

export type AgentStatus =
  | 'idle'
  | 'working'
  | 'paused'
  | 'blocked'
  | 'failed'
  | 'done'
  | 'degraded'

export type IssueStatus =
  | 'pending'
  | 'in_progress'
  | 'blocked'
  | 'done'
  | 'failed'

export type HireStatus = 'pending' | 'approved' | 'rejected'

export type PermissionLevel = 'read' | 'write' | 'admin'

export interface Company {
  id: string
  name: string
  goal: string
  created_at: string
}

export interface Agent {
  id: string
  company_id: string
  role: string
  title: string
  system_prompt: string
  manager_id: string | null
  runtime: AgentRuntime
  status: AgentStatus
  monthly_budget: number
  token_spend: number
  chat_token_spend: number
  pid: number | null
  created_at: string
  updated_at: string
}

export interface Issue {
  id: string
  company_id: string
  title: string
  description: string
  assignee_id: string | null
  parent_id: string | null
  status: IssueStatus
  output_path: string | null
  attempt_count: number
  last_failure_reason: string | null
  escalation_id: string | null
  created_at: string
  updated_at: string
}

export interface EscalationChainEntry {
  agent_id: string
  reason: string
  attempted_at: string
}

export interface Escalation {
  id: string
  original_issue_id: string
  current_assignee_id: string | null
  escalation_chain: EscalationChainEntry[]
  trigger: string
  status: string
  created_at: string
  updated_at: string
}

export interface PendingHire {
  id: string
  company_id: string
  requested_by_agent_id: string
  role_title: string
  reporting_to_agent_id: string
  system_prompt: string
  runtime: AgentRuntime
  budget_allocation: number
  initial_task: string | null
  status: HireStatus
  created_at: string
}

export interface AuditLog {
  id: string
  company_id: string
  actor_id: string | null
  event_type: string
  payload: Record<string, unknown>
  created_at: string
}

export interface Notification {
  id: string
  company_id: string
  type: string
  escalation_id: string | null
  payload: Record<string, unknown>
  dismissed_at: string | null
  created_at: string
}

export interface ChatMessage {
  role: 'user' | 'assistant'
  content: string
  timestamp: string
}

export interface FSEntry {
  name: string
  is_dir: boolean
  size: number
}

export interface FSPermission {
  id: string
  agent_id: string
  path: string
  permission_level: 'read' | 'write' | 'admin'
  granted_by: string | null
}

// WebSocket event envelope (matches ws.Event in Go)
export interface WsEvent {
  type:
    | 'agent_status'
    | 'agent_log'
    | 'issue_update'
    | 'heartbeat'
    | 'notification'
    | 'hire_pending'
    | 'chat_message'
    | 'escalation'
    | 'runtime_status'
  company_id: string
  payload: unknown
}
