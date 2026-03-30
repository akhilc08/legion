package agent

import (
	"context"

	"conductor/internal/store"
	"github.com/google/uuid"
)

// AgentConfig is passed to Spawn to configure a new agent subprocess.
type AgentConfig struct {
	AgentID      uuid.UUID
	CompanyID    uuid.UUID
	SystemPrompt string
	WorkDir      string // shared FS working directory for this agent
	SSHKeyPath   string // path to the agent's SSH private key for SFTP access
	SFTPHost     string
	SFTPPort     int
	EnvVars      map[string]string // additional env vars injected into subprocess
}

// AgentRuntime defines the interface that all agent backends must satisfy.
type AgentRuntime interface {
	// Spawn starts the agent subprocess.
	Spawn(ctx context.Context, config AgentConfig) error

	// SendTask delivers an issue to the running agent.
	SendTask(ctx context.Context, issue store.Issue) error

	// SendChat sends a human chat message and returns the agent's response.
	// Blocks until a full response is received.
	SendChat(ctx context.Context, message string) (string, error)

	// Heartbeat pings the agent to confirm it is alive.
	Heartbeat(ctx context.Context) error

	// Pause signals the agent to finish its current atomic operation
	// and enter interactive chat mode.
	Pause(ctx context.Context) error

	// Resume signals the agent to exit interactive mode and continue its task.
	Resume(ctx context.Context) error

	// Kill terminates the agent subprocess immediately.
	Kill(ctx context.Context) error

	// PID returns the OS process ID, or 0 if not running.
	PID() int

	// TokensUsed returns the total tokens consumed since spawn.
	TokensUsed() int
}

// ControlMessage types emitted by agents on stdout.
const (
	ControlHire      = "CONDUCTOR_HIRE"
	ControlEscalate  = "CONDUCTOR_ESCALATE"
	ControlDone      = "CONDUCTOR_DONE"
	ControlBlocked   = "CONDUCTOR_BLOCKED"
	ControlHeartbeat = "CONDUCTOR_HEARTBEAT"
)

var controlPrefixes = []string{
	ControlHire, ControlEscalate, ControlDone, ControlBlocked, ControlHeartbeat,
}

// HirePayload is the JSON body of a CONDUCTOR_HIRE message.
type HirePayload struct {
	RoleTitle          string       `json:"role_title"`
	ReportingTo        uuid.UUID    `json:"reporting_to"`
	SystemPrompt       string       `json:"system_prompt"`
	Runtime            store.AgentRuntime `json:"runtime"`
	BudgetAllocation   int          `json:"budget_allocation"`
	InitialTask        string       `json:"initial_task,omitempty"`
}

// EscalatePayload is the JSON body of a CONDUCTOR_ESCALATE message.
type EscalatePayload struct {
	IssueID uuid.UUID `json:"issue_id"`
	Reason  string    `json:"reason"`
}

// DonePayload is the JSON body of a CONDUCTOR_DONE message.
type DonePayload struct {
	OutputPath string `json:"output_path"`
	TokensUsed int    `json:"tokens_used"`
}

// BlockedPayload is the JSON body of a CONDUCTOR_BLOCKED message.
type BlockedPayload struct {
	WaitingOnIssueID uuid.UUID `json:"waiting_on_issue_id"`
}
