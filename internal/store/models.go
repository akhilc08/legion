package store

import (
	"time"

	"github.com/google/uuid"
)

type AgentRuntime string

const (
	RuntimeClaudeCode AgentRuntime = "claude_code"
	RuntimeOpenClaw   AgentRuntime = "openclaw"
)

type AgentStatus string

const (
	StatusIdle     AgentStatus = "idle"
	StatusWorking  AgentStatus = "working"
	StatusPaused   AgentStatus = "paused"
	StatusBlocked  AgentStatus = "blocked"
	StatusFailed   AgentStatus = "failed"
	StatusDone     AgentStatus = "done"
	StatusDegraded AgentStatus = "degraded"
)

type Company struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Goal      string    `json:"goal"`
	CreatedAt time.Time `json:"created_at"`
}

type Agent struct {
	ID             uuid.UUID    `json:"id"`
	CompanyID      uuid.UUID    `json:"company_id"`
	Role           string       `json:"role"`
	Title          string       `json:"title"`
	SystemPrompt   string       `json:"system_prompt"`
	ManagerID      *uuid.UUID   `json:"manager_id,omitempty"`
	Runtime        AgentRuntime `json:"runtime"`
	Status         AgentStatus  `json:"status"`
	MonthlyBudget  int          `json:"monthly_budget"`
	TokenSpend     int          `json:"token_spend"`
	ChatTokenSpend int          `json:"chat_token_spend"`
	PID            *int         `json:"pid,omitempty"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
}

type IssueStatus string

const (
	IssuePending    IssueStatus = "pending"
	IssueInProgress IssueStatus = "in_progress"
	IssueBlocked    IssueStatus = "blocked"
	IssueDone       IssueStatus = "done"
	IssueFailed     IssueStatus = "failed"
)

type Issue struct {
	ID                uuid.UUID   `json:"id"`
	CompanyID         uuid.UUID   `json:"company_id"`
	Title             string      `json:"title"`
	Description       string      `json:"description"`
	AssigneeID        *uuid.UUID  `json:"assignee_id,omitempty"`
	ParentID          *uuid.UUID  `json:"parent_id,omitempty"`
	Status            IssueStatus `json:"status"`
	OutputPath        *string     `json:"output_path,omitempty"`
	CreatedBy         *uuid.UUID  `json:"created_by,omitempty"`
	AttemptCount      int         `json:"attempt_count"`
	LastFailureReason *string     `json:"last_failure_reason,omitempty"`
	EscalationID      *uuid.UUID  `json:"escalation_id,omitempty"`
	CreatedAt         time.Time   `json:"created_at"`
	UpdatedAt         time.Time   `json:"updated_at"`
}

type EscalationTrigger string

const (
	TriggerExplicitFailure     EscalationTrigger = "explicit_failure"
	TriggerAttemptLimit        EscalationTrigger = "attempt_limit"
	TriggerBudgetExhausted     EscalationTrigger = "budget_exhausted"
	TriggerDependencyTimeout   EscalationTrigger = "dependency_timeout"
)

type EscalationStatus string

const (
	EscalationOpen            EscalationStatus = "open"
	EscalationResolved        EscalationStatus = "resolved"
	EscalationEscalatedHuman  EscalationStatus = "escalated_to_human"
)

type EscalationChainEntry struct {
	AgentID     uuid.UUID `json:"agent_id"`
	Reason      string    `json:"reason"`
	AttemptedAt time.Time `json:"attempted_at"`
}

type Escalation struct {
	ID               uuid.UUID              `json:"id"`
	OriginalIssueID  uuid.UUID              `json:"original_issue_id"`
	CurrentAssignee  *uuid.UUID             `json:"current_assignee_id,omitempty"`
	EscalationChain  []EscalationChainEntry `json:"escalation_chain"`
	Trigger          EscalationTrigger      `json:"trigger"`
	Status           EscalationStatus       `json:"status"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

type Heartbeat struct {
	AgentID           uuid.UUID `json:"agent_id"`
	LastSeenAt        time.Time `json:"last_seen_at"`
	ConsecutiveMisses int       `json:"consecutive_misses"`
}

type PermissionLevel string

const (
	PermRead  PermissionLevel = "read"
	PermWrite PermissionLevel = "write"
	PermAdmin PermissionLevel = "admin"
)

type FSPermission struct {
	ID              uuid.UUID       `json:"id"`
	AgentID         uuid.UUID       `json:"agent_id"`
	Path            string          `json:"path"`
	PermissionLevel PermissionLevel `json:"permission_level"`
	GrantedBy       *uuid.UUID      `json:"granted_by,omitempty"`
}

type AuditLog struct {
	ID        uuid.UUID              `json:"id"`
	CompanyID uuid.UUID              `json:"company_id"`
	ActorID   *uuid.UUID             `json:"actor_id,omitempty"`
	EventType string                 `json:"event_type"`
	Payload   map[string]interface{} `json:"payload"`
	CreatedAt time.Time              `json:"created_at"`
}

type ChatMessage struct {
	Role      string    `json:"role"` // "user" | "assistant"
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

type ChatSession struct {
	ID        uuid.UUID     `json:"id"`
	AgentID   uuid.UUID     `json:"agent_id"`
	IssueID   *uuid.UUID    `json:"issue_id,omitempty"`
	Messages  []ChatMessage `json:"messages"`
	StartedAt time.Time     `json:"started_at"`
	ResumedAt *time.Time    `json:"resumed_at,omitempty"`
}

type Notification struct {
	ID            uuid.UUID  `json:"id"`
	CompanyID     uuid.UUID  `json:"company_id"`
	Type          string     `json:"type"`
	EscalationID  *uuid.UUID `json:"escalation_id,omitempty"`
	Payload       map[string]interface{} `json:"payload"`
	DismissedAt   *time.Time `json:"dismissed_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

type HireStatus string

const (
	HirePending  HireStatus = "pending"
	HireApproved HireStatus = "approved"
	HireRejected HireStatus = "rejected"
)

type PendingHire struct {
	ID                   uuid.UUID    `json:"id"`
	CompanyID            uuid.UUID    `json:"company_id"`
	RequestedByAgentID   uuid.UUID    `json:"requested_by_agent_id"`
	RoleTitle            string       `json:"role_title"`
	ReportingToAgentID   uuid.UUID    `json:"reporting_to_agent_id"`
	SystemPrompt         string       `json:"system_prompt"`
	Runtime              AgentRuntime `json:"runtime"`
	BudgetAllocation     int          `json:"budget_allocation"`
	InitialTask          *string      `json:"initial_task,omitempty"`
	Status               HireStatus   `json:"status"`
	CreatedAt            time.Time    `json:"created_at"`
}

type User struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}
