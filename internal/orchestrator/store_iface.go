package orchestrator

import (
	"context"
	"time"

	"conductor/internal/store"
	"conductor/internal/ws"
	"github.com/google/uuid"
)

// StoreIface is the subset of *store.DB methods used by the Orchestrator.
// Using an interface makes the orchestrator testable without a real database.
type StoreIface interface {
	// Companies
	ListCompanies(ctx context.Context) ([]store.Company, error)

	// Agents
	ListAgentsByCompany(ctx context.Context, companyID uuid.UUID) ([]store.Agent, error)
	CreateAgent(ctx context.Context, a *store.Agent) (*store.Agent, error)
	GetAgent(ctx context.Context, id uuid.UUID) (*store.Agent, error)
	DeleteAgent(ctx context.Context, id uuid.UUID) error
	UpdateAgentStatus(ctx context.Context, id uuid.UUID, status store.AgentStatus) error
	UpdateAgentPID(ctx context.Context, id uuid.UUID, pid *int) error
	UpdateAgentManager(ctx context.Context, agentID, managerID uuid.UUID) error
	AddTokenSpend(ctx context.Context, agentID uuid.UUID, tokens int, isChat bool) error
	GetSubtreeAgentIDs(ctx context.Context, agentID uuid.UUID) ([]uuid.UUID, error)

	// Issues
	ListIssuesByCompany(ctx context.Context, companyID uuid.UUID) ([]store.Issue, error)
	CreateIssue(ctx context.Context, issue *store.Issue) (*store.Issue, error)
	GetIssue(ctx context.Context, id uuid.UUID) (*store.Issue, error)
	UpdateIssueAssignee(ctx context.Context, id, assigneeID uuid.UUID) error
	UpdateIssueStatus(ctx context.Context, id uuid.UUID, status store.IssueStatus) error
	UpdateIssueOutput(ctx context.Context, id uuid.UUID, outputPath string) error
	IncrementAttemptCount(ctx context.Context, id uuid.UUID, reason string) error
	CheckoutIssue(ctx context.Context, issueID uuid.UUID, agentID uuid.UUID) (bool, error)
	ListReadyIssues(ctx context.Context, companyID uuid.UUID) ([]store.Issue, error)
	AddDependency(ctx context.Context, issueID, dependsOnID uuid.UUID) error

	// Hiring
	CreatePendingHire(ctx context.Context, h *store.PendingHire) (*store.PendingHire, error)
	GetPendingHire(ctx context.Context, id uuid.UUID) (*store.PendingHire, error)
	UpdateHireStatus(ctx context.Context, id uuid.UUID, status store.HireStatus) error

	// Heartbeat
	UpsertHeartbeat(ctx context.Context, agentID uuid.UUID) error
	IncrementMisses(ctx context.Context, agentID uuid.UUID) (int, error)
	StaleAgents(ctx context.Context, threshold time.Duration) ([]store.Agent, error)

	// FS Permissions
	CascadePermissionsFromManager(ctx context.Context, agentID, managerID uuid.UUID) error

	// Notifications
	CreateNotification(ctx context.Context, n *store.Notification) (*store.Notification, error)

	// Audit
	Log(ctx context.Context, companyID uuid.UUID, actorID *uuid.UUID, eventType string, payload map[string]interface{}) error
}

// HubIface is the subset of *ws.Hub methods used by the Orchestrator.
type HubIface interface {
	Broadcast(event ws.Event)
	BroadcastAll(event ws.Event)
}
