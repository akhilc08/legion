package api

import (
	"context"
	"time"

	"conductor/internal/orchestrator"
	"conductor/internal/store"
	"github.com/google/uuid"
)

// Store is the subset of *store.DB methods used by handlers.
type Store interface {
	// Users
	CreateUser(ctx context.Context, email, passwordHash string) (*store.User, error)
	GetUserByEmail(ctx context.Context, email string) (*store.User, error)
	AddUserToCompany(ctx context.Context, userID, companyID uuid.UUID, role string) error
	UserCanAccessCompany(ctx context.Context, userID, companyID uuid.UUID) (bool, error)

	// Companies
	ListCompaniesForUser(ctx context.Context, userID uuid.UUID) ([]store.Company, error)
	CreateCompany(ctx context.Context, name, goal string) (*store.Company, error)
	GetCompany(ctx context.Context, id uuid.UUID) (*store.Company, error)
	UpdateCompanyGoal(ctx context.Context, id uuid.UUID, goal string) error
	DeleteCompany(ctx context.Context, id uuid.UUID) error

	// Agents
	ListAgentsByCompany(ctx context.Context, companyID uuid.UUID) ([]store.Agent, error)
	CreateAgent(ctx context.Context, a *store.Agent) (*store.Agent, error)
	GetAgent(ctx context.Context, id uuid.UUID) (*store.Agent, error)
	DeleteAgent(ctx context.Context, id uuid.UUID) error
	GetCEO(ctx context.Context, companyID uuid.UUID) (*store.Agent, error)

	// Issues
	ListIssuesByCompany(ctx context.Context, companyID uuid.UUID) ([]store.Issue, error)
	CreateIssue(ctx context.Context, issue *store.Issue) (*store.Issue, error)
	GetIssue(ctx context.Context, id uuid.UUID) (*store.Issue, error)
	UpdateIssueAssignee(ctx context.Context, id, assigneeID uuid.UUID) error
	UpdateIssueStatus(ctx context.Context, id uuid.UUID, status store.IssueStatus) error
	AddDependency(ctx context.Context, issueID, dependsOnID uuid.UUID) error
	RemoveDependency(ctx context.Context, issueID, dependsOnID uuid.UUID) error

	// Hiring
	ListPendingHires(ctx context.Context, companyID uuid.UUID) ([]store.PendingHire, error)
	CreatePendingHire(ctx context.Context, h *store.PendingHire) (*store.PendingHire, error)

	// FS Permissions
	ListFSPermissionsForCompany(ctx context.Context, companyID uuid.UUID) ([]store.FSPermission, error)
	GrantFSPermission(ctx context.Context, agentID uuid.UUID, path string, level store.PermissionLevel, grantedBy *uuid.UUID) error
	RevokeFSPermissionByID(ctx context.Context, permID uuid.UUID) error

	// Audit & Notifications
	ListAuditLog(ctx context.Context, companyID uuid.UUID, limit int) ([]store.AuditLog, error)
	ListActiveNotifications(ctx context.Context, companyID uuid.UUID) ([]store.Notification, error)
	DismissNotification(ctx context.Context, id uuid.UUID) error

	// Chat
	GetOrCreateChatSession(ctx context.Context, agentID uuid.UUID) (*store.ChatSession, error)
	AppendChatMessage(ctx context.Context, agentID uuid.UUID, role, content string) error
	GetChatHistory(ctx context.Context, agentID uuid.UUID) ([]store.ChatMessage, error)

	// Heartbeat (used internally but exposed for completeness)
	UpsertHeartbeat(ctx context.Context, agentID uuid.UUID) error

	// Additional methods used by orchestrator internals (via store.DB)
	ListCompanies(ctx context.Context) ([]store.Company, error)
	UpdateAgentStatus(ctx context.Context, id uuid.UUID, status store.AgentStatus) error
	UpdateAgentPID(ctx context.Context, id uuid.UUID, pid *int) error
	UpdateAgentManager(ctx context.Context, agentID, managerID uuid.UUID) error
	AddTokenSpend(ctx context.Context, agentID uuid.UUID, tokens int, isChat bool) error
	GetSubtreeAgentIDs(ctx context.Context, agentID uuid.UUID) ([]uuid.UUID, error)
	ListReadyIssues(ctx context.Context, companyID uuid.UUID) ([]store.Issue, error)
	UpdateIssueOutput(ctx context.Context, id uuid.UUID, outputPath string) error
	IncrementAttemptCount(ctx context.Context, id uuid.UUID, reason string) error
	CheckoutIssue(ctx context.Context, issueID uuid.UUID, agentID uuid.UUID) (bool, error)
	GetPendingHire(ctx context.Context, id uuid.UUID) (*store.PendingHire, error)
	UpdateHireStatus(ctx context.Context, id uuid.UUID, status store.HireStatus) error
	GetHeartbeat(ctx context.Context, agentID uuid.UUID) (*store.Heartbeat, error)
	IncrementMisses(ctx context.Context, agentID uuid.UUID) (int, error)
	StaleAgents(ctx context.Context, threshold time.Duration) ([]store.Agent, error)
	Log(ctx context.Context, companyID uuid.UUID, actorID *uuid.UUID, eventType string, payload map[string]interface{}) error
	CreateNotification(ctx context.Context, n *store.Notification) (*store.Notification, error)
	CascadePermissionsFromManager(ctx context.Context, agentID, managerID uuid.UUID) error
	GetDependencies(ctx context.Context, issueID uuid.UUID) ([]uuid.UUID, error)
	RevokeFSPermission(ctx context.Context, agentID uuid.UUID, path string) error
	ListFSPermissions(ctx context.Context, agentID uuid.UUID) ([]store.FSPermission, error)
}

// Orchestrator is the subset of *orchestrator.Orchestrator methods used by handlers.
type Orchestrator interface {
	AvailableRuntimes() orchestrator.Runtimes
	SpawnAgent(ctx context.Context, agent *store.Agent) error
	KillAgent(ctx context.Context, agentID uuid.UUID) error
	PauseAgent(ctx context.Context, agentID uuid.UUID) error
	ResumeAgent(ctx context.Context, agentID uuid.UUID) error
	ReassignAgent(ctx context.Context, agentID, newManagerID uuid.UUID) error
	SendChatMessage(ctx context.Context, agentID uuid.UUID, message string) (string, error)
	TriggerAssign(agentID uuid.UUID)
	TriggerAssignCompany(ctx context.Context, companyID uuid.UUID)
	ApproveHire(ctx context.Context, hireID uuid.UUID) error
	RejectHire(ctx context.Context, hireID uuid.UUID, reason string) error
}
