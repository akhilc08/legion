package api

import (
	"context"
	"time"

	"conductor/internal/orchestrator"
	"conductor/internal/store"
	"github.com/google/uuid"
)

// ── mockStore ────────────────────────────────────────────────────────────────

type mockStore struct {
	// Users
	createUserFn        func(ctx context.Context, email, hash string) (*store.User, error)
	getUserByEmailFn    func(ctx context.Context, email string) (*store.User, error)
	addUserToCompanyFn  func(ctx context.Context, userID, companyID uuid.UUID, role string) error
	userCanAccessFn     func(ctx context.Context, userID, companyID uuid.UUID) (bool, error)

	// Companies
	listCompaniesForUserFn func(ctx context.Context, userID uuid.UUID) ([]store.Company, error)
	createCompanyFn        func(ctx context.Context, name, goal string) (*store.Company, error)
	getCompanyFn           func(ctx context.Context, id uuid.UUID) (*store.Company, error)
	updateCompanyGoalFn    func(ctx context.Context, id uuid.UUID, goal string) error
	deleteCompanyFn        func(ctx context.Context, id uuid.UUID) error
	listCompaniesFn        func(ctx context.Context) ([]store.Company, error)

	// Agents
	listAgentsByCompanyFn func(ctx context.Context, companyID uuid.UUID) ([]store.Agent, error)
	createAgentFn         func(ctx context.Context, a *store.Agent) (*store.Agent, error)
	getAgentFn            func(ctx context.Context, id uuid.UUID) (*store.Agent, error)
	deleteAgentFn         func(ctx context.Context, id uuid.UUID) error
	getCEOFn              func(ctx context.Context, companyID uuid.UUID) (*store.Agent, error)
	updateAgentStatusFn   func(ctx context.Context, id uuid.UUID, status store.AgentStatus) error
	updateAgentPIDFn      func(ctx context.Context, id uuid.UUID, pid *int) error
	updateAgentManagerFn  func(ctx context.Context, agentID, managerID uuid.UUID) error
	addTokenSpendFn       func(ctx context.Context, agentID uuid.UUID, tokens int, isChat bool) error
	getSubtreeAgentIDsFn  func(ctx context.Context, agentID uuid.UUID) ([]uuid.UUID, error)

	// Issues
	listIssuesByCompanyFn func(ctx context.Context, companyID uuid.UUID) ([]store.Issue, error)
	createIssueFn         func(ctx context.Context, issue *store.Issue) (*store.Issue, error)
	getIssueFn            func(ctx context.Context, id uuid.UUID) (*store.Issue, error)
	updateIssueAssigneeFn func(ctx context.Context, id, assigneeID uuid.UUID) error
	updateIssueStatusFn   func(ctx context.Context, id uuid.UUID, status store.IssueStatus) error
	addDependencyFn       func(ctx context.Context, issueID, dependsOnID uuid.UUID) error
	removeDependencyFn    func(ctx context.Context, issueID, dependsOnID uuid.UUID) error
	listReadyIssuesFn     func(ctx context.Context, companyID uuid.UUID) ([]store.Issue, error)
	updateIssueOutputFn   func(ctx context.Context, id uuid.UUID, outputPath string) error
	incrementAttemptFn    func(ctx context.Context, id uuid.UUID, reason string) error
	checkoutIssueFn       func(ctx context.Context, issueID uuid.UUID, agentID uuid.UUID) (bool, error)
	getDependenciesFn     func(ctx context.Context, issueID uuid.UUID) ([]uuid.UUID, error)

	// Hiring
	listPendingHiresFn func(ctx context.Context, companyID uuid.UUID) ([]store.PendingHire, error)
	createPendingHireFn func(ctx context.Context, h *store.PendingHire) (*store.PendingHire, error)
	getPendingHireFn   func(ctx context.Context, id uuid.UUID) (*store.PendingHire, error)
	updateHireStatusFn func(ctx context.Context, id uuid.UUID, status store.HireStatus) error

	// FS
	listFSPermissionsForCompanyFn func(ctx context.Context, companyID uuid.UUID) ([]store.FSPermission, error)
	grantFSPermissionFn           func(ctx context.Context, agentID uuid.UUID, path string, level store.PermissionLevel, grantedBy *uuid.UUID) error
	revokeFSPermissionByIDFn      func(ctx context.Context, permID uuid.UUID) error
	revokeFSPermissionFn          func(ctx context.Context, agentID uuid.UUID, path string) error
	listFSPermissionsFn           func(ctx context.Context, agentID uuid.UUID) ([]store.FSPermission, error)
	cascadePermissionsFn          func(ctx context.Context, agentID, managerID uuid.UUID) error

	// Audit & Notifications
	listAuditLogFn           func(ctx context.Context, companyID uuid.UUID, limit int) ([]store.AuditLog, error)
	listActiveNotificationsFn func(ctx context.Context, companyID uuid.UUID) ([]store.Notification, error)
	dismissNotificationFn    func(ctx context.Context, id uuid.UUID) error
	createNotificationFn     func(ctx context.Context, n *store.Notification) (*store.Notification, error)
	logFn                    func(ctx context.Context, companyID uuid.UUID, actorID *uuid.UUID, eventType string, payload map[string]interface{}) error

	// Chat
	getOrCreateChatSessionFn func(ctx context.Context, agentID uuid.UUID) (*store.ChatSession, error)
	appendChatMessageFn      func(ctx context.Context, agentID uuid.UUID, role, content string) error
	getChatHistoryFn         func(ctx context.Context, agentID uuid.UUID) ([]store.ChatMessage, error)

	// Heartbeat
	upsertHeartbeatFn  func(ctx context.Context, agentID uuid.UUID) error
	getHeartbeatFn     func(ctx context.Context, agentID uuid.UUID) (*store.Heartbeat, error)
	incrementMissesFn  func(ctx context.Context, agentID uuid.UUID) (int, error)
	staleAgentsFn      func(ctx context.Context, threshold time.Duration) ([]store.Agent, error)
}

func newMockStore() *mockStore {
	errNotImpl := func() error { return nil }
	_ = errNotImpl
	return &mockStore{
		createUserFn:        func(_ context.Context, _, _ string) (*store.User, error) { return nil, errStub },
		getUserByEmailFn:    func(_ context.Context, _ string) (*store.User, error) { return nil, errStub },
		addUserToCompanyFn:  func(_ context.Context, _, _ uuid.UUID, _ string) error { return nil },
		userCanAccessFn:     func(_ context.Context, _, _ uuid.UUID) (bool, error) { return true, nil },

		listCompaniesForUserFn: func(_ context.Context, _ uuid.UUID) ([]store.Company, error) { return nil, errStub },
		createCompanyFn:        func(_ context.Context, _, _ string) (*store.Company, error) { return nil, errStub },
		getCompanyFn:           func(_ context.Context, _ uuid.UUID) (*store.Company, error) { return nil, errStub },
		updateCompanyGoalFn:    func(_ context.Context, _ uuid.UUID, _ string) error { return errStub },
		deleteCompanyFn:        func(_ context.Context, _ uuid.UUID) error { return errStub },
		listCompaniesFn:        func(_ context.Context) ([]store.Company, error) { return nil, errStub },

		listAgentsByCompanyFn: func(_ context.Context, _ uuid.UUID) ([]store.Agent, error) { return nil, errStub },
		createAgentFn:         func(_ context.Context, _ *store.Agent) (*store.Agent, error) { return nil, errStub },
		getAgentFn:            func(_ context.Context, _ uuid.UUID) (*store.Agent, error) { return nil, errStub },
		deleteAgentFn:         func(_ context.Context, _ uuid.UUID) error { return errStub },
		getCEOFn:              func(_ context.Context, _ uuid.UUID) (*store.Agent, error) { return nil, errStub },
		updateAgentStatusFn:   func(_ context.Context, _ uuid.UUID, _ store.AgentStatus) error { return nil },
		updateAgentPIDFn:      func(_ context.Context, _ uuid.UUID, _ *int) error { return nil },
		updateAgentManagerFn:  func(_ context.Context, _, _ uuid.UUID) error { return nil },
		addTokenSpendFn:       func(_ context.Context, _ uuid.UUID, _ int, _ bool) error { return nil },
		getSubtreeAgentIDsFn:  func(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) { return nil, nil },

		listIssuesByCompanyFn: func(_ context.Context, _ uuid.UUID) ([]store.Issue, error) { return nil, errStub },
		createIssueFn:         func(_ context.Context, _ *store.Issue) (*store.Issue, error) { return nil, errStub },
		getIssueFn:            func(_ context.Context, _ uuid.UUID) (*store.Issue, error) { return nil, errStub },
		updateIssueAssigneeFn: func(_ context.Context, _, _ uuid.UUID) error { return errStub },
		updateIssueStatusFn:   func(_ context.Context, _ uuid.UUID, _ store.IssueStatus) error { return errStub },
		addDependencyFn:       func(_ context.Context, _, _ uuid.UUID) error { return errStub },
		removeDependencyFn:    func(_ context.Context, _, _ uuid.UUID) error { return errStub },
		listReadyIssuesFn:     func(_ context.Context, _ uuid.UUID) ([]store.Issue, error) { return nil, nil },
		updateIssueOutputFn:   func(_ context.Context, _ uuid.UUID, _ string) error { return nil },
		incrementAttemptFn:    func(_ context.Context, _ uuid.UUID, _ string) error { return nil },
		checkoutIssueFn:       func(_ context.Context, _, _ uuid.UUID) (bool, error) { return false, nil },
		getDependenciesFn:     func(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) { return nil, nil },

		listPendingHiresFn:  func(_ context.Context, _ uuid.UUID) ([]store.PendingHire, error) { return nil, errStub },
		createPendingHireFn: func(_ context.Context, _ *store.PendingHire) (*store.PendingHire, error) { return nil, errStub },
		getPendingHireFn:    func(_ context.Context, _ uuid.UUID) (*store.PendingHire, error) { return nil, errStub },
		updateHireStatusFn:  func(_ context.Context, _ uuid.UUID, _ store.HireStatus) error { return nil },

		listFSPermissionsForCompanyFn: func(_ context.Context, _ uuid.UUID) ([]store.FSPermission, error) { return nil, errStub },
		grantFSPermissionFn:           func(_ context.Context, _ uuid.UUID, _ string, _ store.PermissionLevel, _ *uuid.UUID) error { return nil },
		revokeFSPermissionByIDFn:      func(_ context.Context, _ uuid.UUID) error { return errStub },
		revokeFSPermissionFn:          func(_ context.Context, _ uuid.UUID, _ string) error { return nil },
		listFSPermissionsFn:           func(_ context.Context, _ uuid.UUID) ([]store.FSPermission, error) { return nil, nil },
		cascadePermissionsFn:          func(_ context.Context, _, _ uuid.UUID) error { return nil },

		listAuditLogFn:            func(_ context.Context, _ uuid.UUID, _ int) ([]store.AuditLog, error) { return nil, errStub },
		listActiveNotificationsFn: func(_ context.Context, _ uuid.UUID) ([]store.Notification, error) { return nil, errStub },
		dismissNotificationFn:    func(_ context.Context, _ uuid.UUID) error { return errStub },
		createNotificationFn:     func(_ context.Context, _ *store.Notification) (*store.Notification, error) { return nil, nil },
		logFn:                    func(_ context.Context, _ uuid.UUID, _ *uuid.UUID, _ string, _ map[string]interface{}) error { return nil },

		getOrCreateChatSessionFn: func(_ context.Context, _ uuid.UUID) (*store.ChatSession, error) { return nil, nil },
		appendChatMessageFn:      func(_ context.Context, _ uuid.UUID, _, _ string) error { return nil },
		getChatHistoryFn:         func(_ context.Context, _ uuid.UUID) ([]store.ChatMessage, error) { return nil, errStub },

		upsertHeartbeatFn: func(_ context.Context, _ uuid.UUID) error { return nil },
		getHeartbeatFn:    func(_ context.Context, _ uuid.UUID) (*store.Heartbeat, error) { return nil, errStub },
		incrementMissesFn: func(_ context.Context, _ uuid.UUID) (int, error) { return 0, nil },
		staleAgentsFn:     func(_ context.Context, _ time.Duration) ([]store.Agent, error) { return nil, nil },
	}
}

// errStub is returned by default mock implementations that haven't been overridden.
var errStub = &stubError{"not configured"}

type stubError struct{ msg string }

func (e *stubError) Error() string { return e.msg }

// Store interface implementation

func (m *mockStore) CreateUser(ctx context.Context, email, hash string) (*store.User, error) {
	return m.createUserFn(ctx, email, hash)
}
func (m *mockStore) GetUserByEmail(ctx context.Context, email string) (*store.User, error) {
	return m.getUserByEmailFn(ctx, email)
}
func (m *mockStore) AddUserToCompany(ctx context.Context, userID, companyID uuid.UUID, role string) error {
	return m.addUserToCompanyFn(ctx, userID, companyID, role)
}
func (m *mockStore) UserCanAccessCompany(ctx context.Context, userID, companyID uuid.UUID) (bool, error) {
	return m.userCanAccessFn(ctx, userID, companyID)
}

func (m *mockStore) ListCompaniesForUser(ctx context.Context, userID uuid.UUID) ([]store.Company, error) {
	return m.listCompaniesForUserFn(ctx, userID)
}
func (m *mockStore) CreateCompany(ctx context.Context, name, goal string) (*store.Company, error) {
	return m.createCompanyFn(ctx, name, goal)
}
func (m *mockStore) GetCompany(ctx context.Context, id uuid.UUID) (*store.Company, error) {
	return m.getCompanyFn(ctx, id)
}
func (m *mockStore) UpdateCompanyGoal(ctx context.Context, id uuid.UUID, goal string) error {
	return m.updateCompanyGoalFn(ctx, id, goal)
}
func (m *mockStore) DeleteCompany(ctx context.Context, id uuid.UUID) error {
	return m.deleteCompanyFn(ctx, id)
}
func (m *mockStore) ListCompanies(ctx context.Context) ([]store.Company, error) {
	return m.listCompaniesFn(ctx)
}

func (m *mockStore) ListAgentsByCompany(ctx context.Context, companyID uuid.UUID) ([]store.Agent, error) {
	return m.listAgentsByCompanyFn(ctx, companyID)
}
func (m *mockStore) CreateAgent(ctx context.Context, a *store.Agent) (*store.Agent, error) {
	return m.createAgentFn(ctx, a)
}
func (m *mockStore) GetAgent(ctx context.Context, id uuid.UUID) (*store.Agent, error) {
	return m.getAgentFn(ctx, id)
}
func (m *mockStore) DeleteAgent(ctx context.Context, id uuid.UUID) error {
	return m.deleteAgentFn(ctx, id)
}
func (m *mockStore) GetCEO(ctx context.Context, companyID uuid.UUID) (*store.Agent, error) {
	return m.getCEOFn(ctx, companyID)
}
func (m *mockStore) UpdateAgentStatus(ctx context.Context, id uuid.UUID, status store.AgentStatus) error {
	return m.updateAgentStatusFn(ctx, id, status)
}
func (m *mockStore) UpdateAgentPID(ctx context.Context, id uuid.UUID, pid *int) error {
	return m.updateAgentPIDFn(ctx, id, pid)
}
func (m *mockStore) UpdateAgentManager(ctx context.Context, agentID, managerID uuid.UUID) error {
	return m.updateAgentManagerFn(ctx, agentID, managerID)
}
func (m *mockStore) AddTokenSpend(ctx context.Context, agentID uuid.UUID, tokens int, isChat bool) error {
	return m.addTokenSpendFn(ctx, agentID, tokens, isChat)
}
func (m *mockStore) GetSubtreeAgentIDs(ctx context.Context, agentID uuid.UUID) ([]uuid.UUID, error) {
	return m.getSubtreeAgentIDsFn(ctx, agentID)
}

func (m *mockStore) ListIssuesByCompany(ctx context.Context, companyID uuid.UUID) ([]store.Issue, error) {
	return m.listIssuesByCompanyFn(ctx, companyID)
}
func (m *mockStore) CreateIssue(ctx context.Context, issue *store.Issue) (*store.Issue, error) {
	return m.createIssueFn(ctx, issue)
}
func (m *mockStore) GetIssue(ctx context.Context, id uuid.UUID) (*store.Issue, error) {
	return m.getIssueFn(ctx, id)
}
func (m *mockStore) UpdateIssueAssignee(ctx context.Context, id, assigneeID uuid.UUID) error {
	return m.updateIssueAssigneeFn(ctx, id, assigneeID)
}
func (m *mockStore) UpdateIssueStatus(ctx context.Context, id uuid.UUID, status store.IssueStatus) error {
	return m.updateIssueStatusFn(ctx, id, status)
}
func (m *mockStore) AddDependency(ctx context.Context, issueID, dependsOnID uuid.UUID) error {
	return m.addDependencyFn(ctx, issueID, dependsOnID)
}
func (m *mockStore) RemoveDependency(ctx context.Context, issueID, dependsOnID uuid.UUID) error {
	return m.removeDependencyFn(ctx, issueID, dependsOnID)
}
func (m *mockStore) ListReadyIssues(ctx context.Context, companyID uuid.UUID) ([]store.Issue, error) {
	return m.listReadyIssuesFn(ctx, companyID)
}
func (m *mockStore) UpdateIssueOutput(ctx context.Context, id uuid.UUID, outputPath string) error {
	return m.updateIssueOutputFn(ctx, id, outputPath)
}
func (m *mockStore) IncrementAttemptCount(ctx context.Context, id uuid.UUID, reason string) error {
	return m.incrementAttemptFn(ctx, id, reason)
}
func (m *mockStore) CheckoutIssue(ctx context.Context, issueID uuid.UUID, agentID uuid.UUID) (bool, error) {
	return m.checkoutIssueFn(ctx, issueID, agentID)
}
func (m *mockStore) GetDependencies(ctx context.Context, issueID uuid.UUID) ([]uuid.UUID, error) {
	return m.getDependenciesFn(ctx, issueID)
}

func (m *mockStore) ListPendingHires(ctx context.Context, companyID uuid.UUID) ([]store.PendingHire, error) {
	return m.listPendingHiresFn(ctx, companyID)
}
func (m *mockStore) CreatePendingHire(ctx context.Context, h *store.PendingHire) (*store.PendingHire, error) {
	return m.createPendingHireFn(ctx, h)
}
func (m *mockStore) GetPendingHire(ctx context.Context, id uuid.UUID) (*store.PendingHire, error) {
	return m.getPendingHireFn(ctx, id)
}
func (m *mockStore) UpdateHireStatus(ctx context.Context, id uuid.UUID, status store.HireStatus) error {
	return m.updateHireStatusFn(ctx, id, status)
}

func (m *mockStore) ListFSPermissionsForCompany(ctx context.Context, companyID uuid.UUID) ([]store.FSPermission, error) {
	return m.listFSPermissionsForCompanyFn(ctx, companyID)
}
func (m *mockStore) GrantFSPermission(ctx context.Context, agentID uuid.UUID, path string, level store.PermissionLevel, grantedBy *uuid.UUID) error {
	return m.grantFSPermissionFn(ctx, agentID, path, level, grantedBy)
}
func (m *mockStore) RevokeFSPermissionByID(ctx context.Context, permID uuid.UUID) error {
	return m.revokeFSPermissionByIDFn(ctx, permID)
}
func (m *mockStore) RevokeFSPermission(ctx context.Context, agentID uuid.UUID, path string) error {
	return m.revokeFSPermissionFn(ctx, agentID, path)
}
func (m *mockStore) ListFSPermissions(ctx context.Context, agentID uuid.UUID) ([]store.FSPermission, error) {
	return m.listFSPermissionsFn(ctx, agentID)
}
func (m *mockStore) CascadePermissionsFromManager(ctx context.Context, agentID, managerID uuid.UUID) error {
	return m.cascadePermissionsFn(ctx, agentID, managerID)
}

func (m *mockStore) ListAuditLog(ctx context.Context, companyID uuid.UUID, limit int) ([]store.AuditLog, error) {
	return m.listAuditLogFn(ctx, companyID, limit)
}
func (m *mockStore) ListActiveNotifications(ctx context.Context, companyID uuid.UUID) ([]store.Notification, error) {
	return m.listActiveNotificationsFn(ctx, companyID)
}
func (m *mockStore) DismissNotification(ctx context.Context, id uuid.UUID) error {
	return m.dismissNotificationFn(ctx, id)
}
func (m *mockStore) CreateNotification(ctx context.Context, n *store.Notification) (*store.Notification, error) {
	return m.createNotificationFn(ctx, n)
}
func (m *mockStore) Log(ctx context.Context, companyID uuid.UUID, actorID *uuid.UUID, eventType string, payload map[string]interface{}) error {
	return m.logFn(ctx, companyID, actorID, eventType, payload)
}

func (m *mockStore) GetOrCreateChatSession(ctx context.Context, agentID uuid.UUID) (*store.ChatSession, error) {
	return m.getOrCreateChatSessionFn(ctx, agentID)
}
func (m *mockStore) AppendChatMessage(ctx context.Context, agentID uuid.UUID, role, content string) error {
	return m.appendChatMessageFn(ctx, agentID, role, content)
}
func (m *mockStore) GetChatHistory(ctx context.Context, agentID uuid.UUID) ([]store.ChatMessage, error) {
	return m.getChatHistoryFn(ctx, agentID)
}

func (m *mockStore) UpsertHeartbeat(ctx context.Context, agentID uuid.UUID) error {
	return m.upsertHeartbeatFn(ctx, agentID)
}
func (m *mockStore) GetHeartbeat(ctx context.Context, agentID uuid.UUID) (*store.Heartbeat, error) {
	return m.getHeartbeatFn(ctx, agentID)
}
func (m *mockStore) IncrementMisses(ctx context.Context, agentID uuid.UUID) (int, error) {
	return m.incrementMissesFn(ctx, agentID)
}
func (m *mockStore) StaleAgents(ctx context.Context, threshold time.Duration) ([]store.Agent, error) {
	return m.staleAgentsFn(ctx, threshold)
}

// ── mockOrch ─────────────────────────────────────────────────────────────────

type mockOrch struct {
	availableRuntimesFn   func() orchestrator.Runtimes
	spawnAgentFn          func(ctx context.Context, agent *store.Agent) error
	killAgentFn           func(ctx context.Context, agentID uuid.UUID) error
	pauseAgentFn          func(ctx context.Context, agentID uuid.UUID) error
	resumeAgentFn         func(ctx context.Context, agentID uuid.UUID) error
	reassignAgentFn       func(ctx context.Context, agentID, newManagerID uuid.UUID) error
	sendChatMessageFn     func(ctx context.Context, agentID uuid.UUID, message string) (string, error)
	triggerAssignFn       func(agentID uuid.UUID)
	triggerAssignCompanyFn func(ctx context.Context, companyID uuid.UUID)
	approveHireFn         func(ctx context.Context, hireID uuid.UUID) error
	rejectHireFn          func(ctx context.Context, hireID uuid.UUID, reason string) error
}

func newMockOrch() *mockOrch {
	return &mockOrch{
		availableRuntimesFn:    func() orchestrator.Runtimes { return orchestrator.Runtimes{} },
		spawnAgentFn:           func(_ context.Context, _ *store.Agent) error { return nil },
		killAgentFn:            func(_ context.Context, _ uuid.UUID) error { return nil },
		pauseAgentFn:           func(_ context.Context, _ uuid.UUID) error { return nil },
		resumeAgentFn:          func(_ context.Context, _ uuid.UUID) error { return nil },
		reassignAgentFn:        func(_ context.Context, _, _ uuid.UUID) error { return nil },
		sendChatMessageFn:      func(_ context.Context, _ uuid.UUID, _ string) (string, error) { return "", errStub },
		triggerAssignFn:        func(_ uuid.UUID) {},
		triggerAssignCompanyFn: func(_ context.Context, _ uuid.UUID) {},
		approveHireFn:          func(_ context.Context, _ uuid.UUID) error { return errStub },
		rejectHireFn:           func(_ context.Context, _ uuid.UUID, _ string) error { return errStub },
	}
}

func (m *mockOrch) AvailableRuntimes() orchestrator.Runtimes {
	return m.availableRuntimesFn()
}
func (m *mockOrch) SpawnAgent(ctx context.Context, agent *store.Agent) error {
	return m.spawnAgentFn(ctx, agent)
}
func (m *mockOrch) KillAgent(ctx context.Context, agentID uuid.UUID) error {
	return m.killAgentFn(ctx, agentID)
}
func (m *mockOrch) PauseAgent(ctx context.Context, agentID uuid.UUID) error {
	return m.pauseAgentFn(ctx, agentID)
}
func (m *mockOrch) ResumeAgent(ctx context.Context, agentID uuid.UUID) error {
	return m.resumeAgentFn(ctx, agentID)
}
func (m *mockOrch) ReassignAgent(ctx context.Context, agentID, newManagerID uuid.UUID) error {
	return m.reassignAgentFn(ctx, agentID, newManagerID)
}
func (m *mockOrch) SendChatMessage(ctx context.Context, agentID uuid.UUID, message string) (string, error) {
	return m.sendChatMessageFn(ctx, agentID, message)
}
func (m *mockOrch) TriggerAssign(agentID uuid.UUID) {
	m.triggerAssignFn(agentID)
}
func (m *mockOrch) TriggerAssignCompany(ctx context.Context, companyID uuid.UUID) {
	m.triggerAssignCompanyFn(ctx, companyID)
}
func (m *mockOrch) ApproveHire(ctx context.Context, hireID uuid.UUID) error {
	return m.approveHireFn(ctx, hireID)
}
func (m *mockOrch) RejectHire(ctx context.Context, hireID uuid.UUID, reason string) error {
	return m.rejectHireFn(ctx, hireID, reason)
}
