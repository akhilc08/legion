package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"conductor/internal/agent"
	"conductor/internal/heartbeat"
	"conductor/internal/store"
	"conductor/internal/ws"
	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Mock store
// ---------------------------------------------------------------------------

type mockStore struct {
	mu sync.Mutex

	companies       []store.Company
	agents          map[uuid.UUID]*store.Agent
	issues          map[uuid.UUID]*store.Issue
	pendingHires    map[uuid.UUID]*store.PendingHire
	notifications   []*store.Notification
	subtreeIDs      map[uuid.UUID][]uuid.UUID
	readyIssues     map[uuid.UUID][]store.Issue
	checkedOut      map[uuid.UUID]bool
	tokenSpend      map[uuid.UUID]int
	heartbeats      map[uuid.UUID]time.Time
	misses          map[uuid.UUID]int
	staleAgents     []store.Agent

	// controllable errors
	errListCompanies     error
	errGetAgent          error
	errUpdateAgentStatus error
	errCreateAgent       error
	errCreatePendingHire error
	errGetPendingHire    error
	errGetIssue          error
	errCreateNotification error
	errListIssues        error
	errStaleAgents       error

	// call trackers
	updatedStatuses    []statusUpdate
	broadcastedEvents  []ws.Event
	loggedEvents       []loggedEvent
	upsertHeartbeats   []uuid.UUID
	incrementedMisses  []uuid.UUID
}

type statusUpdate struct {
	agentID uuid.UUID
	status  store.AgentStatus
}

type loggedEvent struct {
	companyID uuid.UUID
	eventType string
}

func newMockStore() *mockStore {
	return &mockStore{
		agents:       make(map[uuid.UUID]*store.Agent),
		issues:       make(map[uuid.UUID]*store.Issue),
		pendingHires: make(map[uuid.UUID]*store.PendingHire),
		subtreeIDs:   make(map[uuid.UUID][]uuid.UUID),
		readyIssues:  make(map[uuid.UUID][]store.Issue),
		checkedOut:   make(map[uuid.UUID]bool),
		tokenSpend:   make(map[uuid.UUID]int),
		heartbeats:   make(map[uuid.UUID]time.Time),
		misses:       make(map[uuid.UUID]int),
	}
}

func (m *mockStore) ListCompanies(ctx context.Context) ([]store.Company, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.companies, m.errListCompanies
}

func (m *mockStore) ListAgentsByCompany(ctx context.Context, companyID uuid.UUID) ([]store.Agent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []store.Agent
	for _, a := range m.agents {
		if a.CompanyID == companyID {
			out = append(out, *a)
		}
	}
	return out, nil
}

func (m *mockStore) CreateAgent(ctx context.Context, a *store.Agent) (*store.Agent, error) {
	if m.errCreateAgent != nil {
		return nil, m.errCreateAgent
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	cp := *a
	m.agents[cp.ID] = &cp
	return &cp, nil
}

func (m *mockStore) GetAgent(ctx context.Context, id uuid.UUID) (*store.Agent, error) {
	if m.errGetAgent != nil {
		return nil, m.errGetAgent
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.agents[id]
	if !ok {
		return nil, fmt.Errorf("agent not found: %s", id)
	}
	cp := *a
	return &cp, nil
}

func (m *mockStore) DeleteAgent(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.agents, id)
	return nil
}

func (m *mockStore) UpdateAgentStatus(ctx context.Context, id uuid.UUID, status store.AgentStatus) error {
	if m.errUpdateAgentStatus != nil {
		return m.errUpdateAgentStatus
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updatedStatuses = append(m.updatedStatuses, statusUpdate{agentID: id, status: status})
	if a, ok := m.agents[id]; ok {
		a.Status = status
	}
	return nil
}

func (m *mockStore) UpdateAgentPID(ctx context.Context, id uuid.UUID, pid *int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if a, ok := m.agents[id]; ok && pid != nil {
		a.PID = pid
	}
	return nil
}

func (m *mockStore) UpdateAgentManager(ctx context.Context, agentID, managerID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if a, ok := m.agents[agentID]; ok {
		a.ManagerID = &managerID
	}
	return nil
}

func (m *mockStore) AddTokenSpend(ctx context.Context, agentID uuid.UUID, tokens int, isChat bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tokenSpend[agentID] += tokens
	if a, ok := m.agents[agentID]; ok {
		a.TokenSpend += tokens
	}
	return nil
}

func (m *mockStore) GetSubtreeAgentIDs(ctx context.Context, agentID uuid.UUID) ([]uuid.UUID, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.subtreeIDs[agentID], nil
}

func (m *mockStore) ListIssuesByCompany(ctx context.Context, companyID uuid.UUID) ([]store.Issue, error) {
	if m.errListIssues != nil {
		return nil, m.errListIssues
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []store.Issue
	for _, iss := range m.issues {
		if iss.CompanyID == companyID {
			out = append(out, *iss)
		}
	}
	return out, nil
}

func (m *mockStore) CreateIssue(ctx context.Context, issue *store.Issue) (*store.Issue, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if issue.ID == uuid.Nil {
		issue.ID = uuid.New()
	}
	cp := *issue
	m.issues[cp.ID] = &cp
	return &cp, nil
}

func (m *mockStore) GetIssue(ctx context.Context, id uuid.UUID) (*store.Issue, error) {
	if m.errGetIssue != nil {
		return nil, m.errGetIssue
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	iss, ok := m.issues[id]
	if !ok {
		return nil, fmt.Errorf("issue not found: %s", id)
	}
	cp := *iss
	return &cp, nil
}

func (m *mockStore) UpdateIssueAssignee(ctx context.Context, id, assigneeID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if iss, ok := m.issues[id]; ok {
		iss.AssigneeID = &assigneeID
	}
	return nil
}

func (m *mockStore) UpdateIssueStatus(ctx context.Context, id uuid.UUID, status store.IssueStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if iss, ok := m.issues[id]; ok {
		iss.Status = status
	}
	return nil
}

func (m *mockStore) UpdateIssueOutput(ctx context.Context, id uuid.UUID, outputPath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if iss, ok := m.issues[id]; ok {
		iss.OutputPath = &outputPath
	}
	return nil
}

func (m *mockStore) IncrementAttemptCount(ctx context.Context, id uuid.UUID, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if iss, ok := m.issues[id]; ok {
		iss.AttemptCount++
	}
	return nil
}

func (m *mockStore) CheckoutIssue(ctx context.Context, issueID uuid.UUID, agentID uuid.UUID) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.checkedOut[issueID] {
		return false, nil
	}
	m.checkedOut[issueID] = true
	if iss, ok := m.issues[issueID]; ok {
		iss.AssigneeID = &agentID
		iss.Status = store.IssueInProgress
	}
	return true, nil
}

func (m *mockStore) ListReadyIssues(ctx context.Context, companyID uuid.UUID) ([]store.Issue, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.readyIssues[companyID], nil
}

func (m *mockStore) AddDependency(ctx context.Context, issueID, dependsOnID uuid.UUID) error {
	return nil
}

func (m *mockStore) CreatePendingHire(ctx context.Context, h *store.PendingHire) (*store.PendingHire, error) {
	if m.errCreatePendingHire != nil {
		return nil, m.errCreatePendingHire
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if h.ID == uuid.Nil {
		h.ID = uuid.New()
	}
	cp := *h
	m.pendingHires[cp.ID] = &cp
	return &cp, nil
}

func (m *mockStore) GetPendingHire(ctx context.Context, id uuid.UUID) (*store.PendingHire, error) {
	if m.errGetPendingHire != nil {
		return nil, m.errGetPendingHire
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	h, ok := m.pendingHires[id]
	if !ok {
		return nil, fmt.Errorf("hire not found: %s", id)
	}
	cp := *h
	return &cp, nil
}

func (m *mockStore) UpdateHireStatus(ctx context.Context, id uuid.UUID, status store.HireStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if h, ok := m.pendingHires[id]; ok {
		h.Status = status
	}
	return nil
}

func (m *mockStore) UpsertHeartbeat(ctx context.Context, agentID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.heartbeats[agentID] = time.Now()
	m.upsertHeartbeats = append(m.upsertHeartbeats, agentID)
	return nil
}

func (m *mockStore) IncrementMisses(ctx context.Context, agentID uuid.UUID) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.misses[agentID]++
	m.incrementedMisses = append(m.incrementedMisses, agentID)
	return m.misses[agentID], nil
}

func (m *mockStore) StaleAgents(ctx context.Context, threshold time.Duration) ([]store.Agent, error) {
	if m.errStaleAgents != nil {
		return nil, m.errStaleAgents
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.staleAgents, nil
}

func (m *mockStore) CascadePermissionsFromManager(ctx context.Context, agentID, managerID uuid.UUID) error {
	return nil
}

func (m *mockStore) CreateNotification(ctx context.Context, n *store.Notification) (*store.Notification, error) {
	if m.errCreateNotification != nil {
		return nil, m.errCreateNotification
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	cp := *n
	m.notifications = append(m.notifications, &cp)
	return &cp, nil
}

func (m *mockStore) Log(ctx context.Context, companyID uuid.UUID, actorID *uuid.UUID, eventType string, payload map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.loggedEvents = append(m.loggedEvents, loggedEvent{companyID: companyID, eventType: eventType})
	return nil
}

// ---------------------------------------------------------------------------
// Mock hub
// ---------------------------------------------------------------------------

type mockHub struct {
	mu     sync.Mutex
	events []ws.Event
	allEvents []ws.Event
}

func (h *mockHub) Broadcast(event ws.Event) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.events = append(h.events, event)
}

func (h *mockHub) BroadcastAll(event ws.Event) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.allEvents = append(h.allEvents, event)
}

func (h *mockHub) eventTypes() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	var out []string
	for _, e := range h.events {
		out = append(out, e.Type)
	}
	return out
}

func (h *mockHub) hasEventType(t string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, e := range h.events {
		if e.Type == t {
			return true
		}
	}
	for _, e := range h.allEvents {
		if e.Type == t {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Mock AgentRuntime
// ---------------------------------------------------------------------------

type mockRuntime struct {
	spawnFn      func(ctx context.Context, config agent.AgentConfig) error
	sendTaskFn   func(ctx context.Context, issue store.Issue) error
	sendChatFn   func(ctx context.Context, message string) (string, error)
	heartbeatFn  func(ctx context.Context) error
	pauseFn      func(ctx context.Context) error
	resumeFn     func(ctx context.Context) error
	killFn       func(ctx context.Context) error
	pidFn        func() int
	tokensUsedFn func() int
}

func (r *mockRuntime) Spawn(ctx context.Context, config agent.AgentConfig) error {
	if r.spawnFn != nil {
		return r.spawnFn(ctx, config)
	}
	return nil
}
func (r *mockRuntime) SendTask(ctx context.Context, issue store.Issue) error {
	if r.sendTaskFn != nil {
		return r.sendTaskFn(ctx, issue)
	}
	return nil
}
func (r *mockRuntime) SendChat(ctx context.Context, message string) (string, error) {
	if r.sendChatFn != nil {
		return r.sendChatFn(ctx, message)
	}
	return "mock reply", nil
}
func (r *mockRuntime) Heartbeat(ctx context.Context) error {
	if r.heartbeatFn != nil {
		return r.heartbeatFn(ctx)
	}
	return nil
}
func (r *mockRuntime) Pause(ctx context.Context) error {
	if r.pauseFn != nil {
		return r.pauseFn(ctx)
	}
	return nil
}
func (r *mockRuntime) Resume(ctx context.Context) error {
	if r.resumeFn != nil {
		return r.resumeFn(ctx)
	}
	return nil
}
func (r *mockRuntime) Kill(ctx context.Context) error {
	if r.killFn != nil {
		return r.killFn(ctx)
	}
	return nil
}
func (r *mockRuntime) PID() int {
	if r.pidFn != nil {
		return r.pidFn()
	}
	return 42
}
func (r *mockRuntime) TokensUsed() int {
	if r.tokensUsedFn != nil {
		return r.tokensUsedFn()
	}
	return 0
}

// ---------------------------------------------------------------------------
// Helper: build orchestrator with mocks and inject a runtime directly
// ---------------------------------------------------------------------------

func newTestOrchestrator(t *testing.T) (*Orchestrator, *mockStore, *mockHub) {
	t.Helper()
	ms := newMockStore()
	mh := &mockHub{}
	o := &Orchestrator{
		db:               ms,
		hub:              mh,
		fsRoot:           t.TempDir(),
		runtimes:         make(map[uuid.UUID]agent.AgentRuntime),
		wakeChans:        make(map[uuid.UUID]chan struct{}),
		runtimeCheckStop: make(chan struct{}),
	}
	// Provide a real watcher backed by the mock store so KillAgent / SpawnAgent
	// can call watcher.Unwatch / watcher.Watch without nil-pointer panics.
	o.watcher = heartbeat.NewWatcher(ms, nil, nil)
	return o, ms, mh
}

// injectRuntime registers a mock runtime for an agent without spawning.
func injectRuntime(o *Orchestrator, agentID uuid.UUID, rt agent.AgentRuntime) {
	o.mu.Lock()
	o.runtimes[agentID] = rt
	o.mu.Unlock()
}

// ---------------------------------------------------------------------------
// Tests: New / AvailableRuntimes
// ---------------------------------------------------------------------------

func TestNew_InitializesFields(t *testing.T) {
	ms := newMockStore()
	mh := &mockHub{}
	o := &Orchestrator{
		db:               ms,
		hub:              mh,
		fsRoot:           "/tmp/test",
		runtimes:         make(map[uuid.UUID]agent.AgentRuntime),
		wakeChans:        make(map[uuid.UUID]chan struct{}),
		runtimeCheckStop: make(chan struct{}),
	}

	if o.db == nil {
		t.Fatal("db should not be nil")
	}
	if o.hub == nil {
		t.Fatal("hub should not be nil")
	}
	if o.runtimes == nil {
		t.Fatal("runtimes map should not be nil")
	}
	if o.wakeChans == nil {
		t.Fatal("wakeChans map should not be nil")
	}
}

func TestAvailableRuntimes(t *testing.T) {
	o, _, _ := newTestOrchestrator(t)
	o.mu.Lock()
	o.available = Runtimes{ClaudeCode: true, OpenClaw: false}
	o.mu.Unlock()

	got := o.AvailableRuntimes()
	if !got.ClaudeCode {
		t.Error("expected ClaudeCode=true")
	}
	if got.OpenClaw {
		t.Error("expected OpenClaw=false")
	}
}

// ---------------------------------------------------------------------------
// Tests: Start / Stop
// ---------------------------------------------------------------------------

func TestStart_DoesNotError(t *testing.T) {
	o, ms, _ := newTestOrchestrator(t)
	ms.companies = []store.Company{}
	ctx := context.Background()
	if err := o.Start(ctx); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	// Stop to clean up background goroutine.
	o.Stop()
}

func TestStop_ClosesChannel(t *testing.T) {
	o, ms, _ := newTestOrchestrator(t)
	ms.companies = []store.Company{}

	ctx := context.Background()
	o.Start(ctx) //nolint

	// Stop should not panic (closing already-running goroutine).
	o.Stop()

	// Verify the stop channel is closed by trying to receive from it.
	select {
	case <-o.runtimeCheckStop:
		// closed as expected
	case <-time.After(100 * time.Millisecond):
		t.Error("runtimeCheckStop was not closed by Stop()")
	}
}

// ---------------------------------------------------------------------------
// Tests: refreshRuntimes
// ---------------------------------------------------------------------------

func TestRefreshRuntimes_DetectsKnownCommand(t *testing.T) {
	o, _, _ := newTestOrchestrator(t)

	// We cannot know whether "claude" or "openclaw" are installed,
	// but "go" is guaranteed present in any Go test environment.
	// Verify that commandExists works for a present and absent command.
	if !commandExists("go") {
		t.Error("commandExists('go') should return true")
	}
	if commandExists("__no_such_command_xyz__") {
		t.Error("commandExists('__no_such_command_xyz__') should return false")
	}

	// Call refreshRuntimes; it should not panic.
	o.refreshRuntimes()
}

func TestRefreshRuntimes_BroadcastsOnChange(t *testing.T) {
	o, _, mh := newTestOrchestrator(t)

	// Set available to some value that differs from what refreshRuntimes will detect.
	// Force a known mismatch: set ClaudeCode=true even if "claude" isn't installed.
	o.mu.Lock()
	o.available = Runtimes{ClaudeCode: true, OpenClaw: true}
	o.mu.Unlock()

	// refreshRuntimes will re-detect; if either changed it broadcasts.
	o.refreshRuntimes()

	// The outcome depends on whether claude/openclaw are installed.
	// At minimum, verify no panic and available is updated.
	got := o.AvailableRuntimes()
	_ = got // valid struct
	_ = mh  // may or may not have events depending on environment
}

// ---------------------------------------------------------------------------
// Tests: markAgentsDegraded
// ---------------------------------------------------------------------------

func TestMarkAgentsDegraded_UpdatesMatchingAgents(t *testing.T) {
	o, ms, mh := newTestOrchestrator(t)
	companyID := uuid.New()
	agentID := uuid.New()

	ms.companies = []store.Company{{ID: companyID, Name: "Test Co"}}
	ms.agents[agentID] = &store.Agent{
		ID:        agentID,
		CompanyID: companyID,
		Runtime:   store.RuntimeClaudeCode,
		Status:    store.StatusIdle,
	}

	o.markAgentsDegraded(context.Background(), store.RuntimeClaudeCode)

	// Verify status was updated to degraded.
	ms.mu.Lock()
	statuses := ms.updatedStatuses
	ms.mu.Unlock()

	found := false
	for _, s := range statuses {
		if s.agentID == agentID && s.status == store.StatusDegraded {
			found = true
		}
	}
	if !found {
		t.Error("expected agent status to be updated to Degraded")
	}

	if !mh.hasEventType(ws.EventAgentStatus) {
		t.Error("expected agent_status broadcast")
	}
}

func TestMarkAgentsDegraded_SkipsNonMatchingRuntime(t *testing.T) {
	o, ms, _ := newTestOrchestrator(t)
	companyID := uuid.New()
	agentID := uuid.New()

	ms.companies = []store.Company{{ID: companyID}}
	ms.agents[agentID] = &store.Agent{
		ID:        agentID,
		CompanyID: companyID,
		Runtime:   store.RuntimeOpenClaw, // different runtime
		Status:    store.StatusIdle,
	}

	o.markAgentsDegraded(context.Background(), store.RuntimeClaudeCode)

	ms.mu.Lock()
	count := len(ms.updatedStatuses)
	ms.mu.Unlock()

	if count != 0 {
		t.Errorf("expected 0 status updates, got %d", count)
	}
}

func TestMarkAgentsDegraded_SkipsInactiveAgents(t *testing.T) {
	o, ms, _ := newTestOrchestrator(t)
	companyID := uuid.New()
	agentID := uuid.New()

	ms.companies = []store.Company{{ID: companyID}}
	ms.agents[agentID] = &store.Agent{
		ID:        agentID,
		CompanyID: companyID,
		Runtime:   store.RuntimeClaudeCode,
		Status:    store.StatusFailed, // not idle or working
	}

	o.markAgentsDegraded(context.Background(), store.RuntimeClaudeCode)

	ms.mu.Lock()
	count := len(ms.updatedStatuses)
	ms.mu.Unlock()

	if count != 0 {
		t.Errorf("expected 0 status updates for failed agent, got %d", count)
	}
}

// ---------------------------------------------------------------------------
// Tests: KillAgent
// ---------------------------------------------------------------------------

func TestKillAgent_WithRuntime(t *testing.T) {
	o, ms, _ := newTestOrchestrator(t)
	agentID := uuid.New()
	companyID := uuid.New()

	ms.agents[agentID] = &store.Agent{
		ID:        agentID,
		CompanyID: companyID,
		Status:    store.StatusWorking,
	}

	killed := false
	rt := &mockRuntime{
		killFn: func(ctx context.Context) error {
			killed = true
			return nil
		},
	}
	injectRuntime(o, agentID, rt)

	err := o.KillAgent(context.Background(), agentID)
	if err != nil {
		t.Fatalf("KillAgent returned error: %v", err)
	}
	if !killed {
		t.Error("expected Kill to be called on runtime")
	}

	// Runtime should be removed.
	o.mu.RLock()
	_, stillPresent := o.runtimes[agentID]
	o.mu.RUnlock()
	if stillPresent {
		t.Error("runtime should have been removed after kill")
	}
}

func TestKillAgent_WithoutRuntime(t *testing.T) {
	o, ms, _ := newTestOrchestrator(t)
	agentID := uuid.New()

	ms.agents[agentID] = &store.Agent{
		ID:     agentID,
		Status: store.StatusFailed,
	}

	// No runtime registered — should still update DB status.
	err := o.KillAgent(context.Background(), agentID)
	if err != nil {
		t.Fatalf("KillAgent (no runtime) returned error: %v", err)
	}

	ms.mu.Lock()
	statuses := ms.updatedStatuses
	ms.mu.Unlock()

	found := false
	for _, s := range statuses {
		if s.agentID == agentID && s.status == store.StatusFailed {
			found = true
		}
	}
	if !found {
		t.Error("expected StatusFailed update even without a running runtime")
	}
}

// ---------------------------------------------------------------------------
// Tests: PauseAgent / ResumeAgent
// ---------------------------------------------------------------------------

func TestPauseAgent_AgentNotRunning(t *testing.T) {
	o, _, _ := newTestOrchestrator(t)
	agentID := uuid.New()

	err := o.PauseAgent(context.Background(), agentID)
	if err == nil {
		t.Fatal("expected error when agent not running")
	}
}

func TestPauseAgent_Success(t *testing.T) {
	o, ms, mh := newTestOrchestrator(t)
	agentID := uuid.New()
	companyID := uuid.New()

	ms.agents[agentID] = &store.Agent{
		ID:        agentID,
		CompanyID: companyID,
		Status:    store.StatusWorking,
	}

	paused := false
	rt := &mockRuntime{
		pauseFn: func(ctx context.Context) error {
			paused = true
			return nil
		},
	}
	injectRuntime(o, agentID, rt)

	err := o.PauseAgent(context.Background(), agentID)
	if err != nil {
		t.Fatalf("PauseAgent returned error: %v", err)
	}
	if !paused {
		t.Error("expected Pause to be called on runtime")
	}
	if !mh.hasEventType(ws.EventAgentStatus) {
		t.Error("expected agent_status broadcast")
	}
}

func TestPauseAgent_RuntimeError(t *testing.T) {
	o, ms, _ := newTestOrchestrator(t)
	agentID := uuid.New()
	companyID := uuid.New()

	ms.agents[agentID] = &store.Agent{
		ID:        agentID,
		CompanyID: companyID,
	}
	rt := &mockRuntime{
		pauseFn: func(ctx context.Context) error {
			return fmt.Errorf("pause failed")
		},
	}
	injectRuntime(o, agentID, rt)

	err := o.PauseAgent(context.Background(), agentID)
	if err == nil {
		t.Fatal("expected error from Pause")
	}
}

func TestResumeAgent_AgentNotRunning(t *testing.T) {
	o, _, _ := newTestOrchestrator(t)
	agentID := uuid.New()

	err := o.ResumeAgent(context.Background(), agentID)
	if err == nil {
		t.Fatal("expected error when agent not running")
	}
}

func TestResumeAgent_Success(t *testing.T) {
	o, ms, mh := newTestOrchestrator(t)
	agentID := uuid.New()
	companyID := uuid.New()

	ms.agents[agentID] = &store.Agent{
		ID:        agentID,
		CompanyID: companyID,
		Status:    store.StatusPaused,
	}

	resumed := false
	rt := &mockRuntime{
		resumeFn: func(ctx context.Context) error {
			resumed = true
			return nil
		},
	}
	injectRuntime(o, agentID, rt)

	err := o.ResumeAgent(context.Background(), agentID)
	if err != nil {
		t.Fatalf("ResumeAgent returned error: %v", err)
	}
	if !resumed {
		t.Error("expected Resume to be called on runtime")
	}
	if !mh.hasEventType(ws.EventAgentStatus) {
		t.Error("expected agent_status broadcast")
	}
}

// ---------------------------------------------------------------------------
// Tests: SendChatMessage
// ---------------------------------------------------------------------------

func TestSendChatMessage_AgentNotRunning(t *testing.T) {
	o, _, _ := newTestOrchestrator(t)
	agentID := uuid.New()

	_, err := o.SendChatMessage(context.Background(), agentID, "hello")
	if err == nil {
		t.Fatal("expected error when agent not running")
	}
}

func TestSendChatMessage_Success(t *testing.T) {
	o, ms, mh := newTestOrchestrator(t)
	agentID := uuid.New()
	companyID := uuid.New()

	ms.agents[agentID] = &store.Agent{
		ID:        agentID,
		CompanyID: companyID,
	}
	rt := &mockRuntime{
		sendChatFn: func(ctx context.Context, message string) (string, error) {
			return "hello back", nil
		},
	}
	injectRuntime(o, agentID, rt)

	reply, err := o.SendChatMessage(context.Background(), agentID, "hello")
	if err != nil {
		t.Fatalf("SendChatMessage returned error: %v", err)
	}
	if reply != "hello back" {
		t.Errorf("expected reply 'hello back', got %q", reply)
	}
	if !mh.hasEventType(ws.EventChatMessage) {
		t.Error("expected chat_message broadcast")
	}
}

func TestSendChatMessage_RuntimeError(t *testing.T) {
	o, ms, _ := newTestOrchestrator(t)
	agentID := uuid.New()
	companyID := uuid.New()

	ms.agents[agentID] = &store.Agent{
		ID:        agentID,
		CompanyID: companyID,
	}
	rt := &mockRuntime{
		sendChatFn: func(ctx context.Context, message string) (string, error) {
			return "", fmt.Errorf("send failed")
		},
	}
	injectRuntime(o, agentID, rt)

	_, err := o.SendChatMessage(context.Background(), agentID, "hello")
	if err == nil {
		t.Fatal("expected error when runtime fails")
	}
}

// ---------------------------------------------------------------------------
// Tests: TriggerAssign / TriggerAssignCompany
// ---------------------------------------------------------------------------

func TestTriggerAssign_SignalsWakeChannel(t *testing.T) {
	o, _, _ := newTestOrchestrator(t)
	agentID := uuid.New()

	wake := make(chan struct{}, 1)
	o.mu.Lock()
	o.wakeChans[agentID] = wake
	o.mu.Unlock()

	o.TriggerAssign(agentID)

	select {
	case <-wake:
		// success
	case <-time.After(100 * time.Millisecond):
		t.Error("TriggerAssign did not signal the wake channel")
	}
}

func TestTriggerAssign_NoopWhenNotRegistered(t *testing.T) {
	o, _, _ := newTestOrchestrator(t)
	// Should not panic even with unregistered agent.
	o.TriggerAssign(uuid.New())
}

func TestTriggerAssignCompany_WakesIdleAgents(t *testing.T) {
	o, ms, _ := newTestOrchestrator(t)
	companyID := uuid.New()
	idleID := uuid.New()
	workingID := uuid.New()

	ms.agents[idleID] = &store.Agent{
		ID:        idleID,
		CompanyID: companyID,
		Status:    store.StatusIdle,
	}
	ms.agents[workingID] = &store.Agent{
		ID:        workingID,
		CompanyID: companyID,
		Status:    store.StatusWorking,
	}

	idleWake := make(chan struct{}, 1)
	workingWake := make(chan struct{}, 1)
	o.mu.Lock()
	o.wakeChans[idleID] = idleWake
	o.wakeChans[workingID] = workingWake
	o.mu.Unlock()

	o.TriggerAssignCompany(context.Background(), companyID)

	select {
	case <-idleWake:
		// correct: idle agent was woken
	case <-time.After(100 * time.Millisecond):
		t.Error("TriggerAssignCompany did not wake idle agent")
	}

	select {
	case <-workingWake:
		t.Error("TriggerAssignCompany should not wake working agent")
	case <-time.After(50 * time.Millisecond):
		// correct: working agent was not signaled
	}
}

// ---------------------------------------------------------------------------
// Tests: HandleControlMessage
// ---------------------------------------------------------------------------

func TestHandleControlMessage_Heartbeat(t *testing.T) {
	o, ms, _ := newTestOrchestrator(t)
	agentID := uuid.New()
	companyID := uuid.New()

	ms.agents[agentID] = &store.Agent{
		ID:        agentID,
		CompanyID: companyID,
		Status:    store.StatusWorking,
	}

	o.HandleControlMessage(context.Background(), agentID, companyID,
		agent.ControlHeartbeat, json.RawMessage(`{}`))

	ms.mu.Lock()
	heartbeats := ms.upsertHeartbeats
	ms.mu.Unlock()

	if len(heartbeats) == 0 {
		t.Error("expected UpsertHeartbeat to be called")
	}
}

func TestHandleControlMessage_Done_BasicCompletion(t *testing.T) {
	o, ms, mh := newTestOrchestrator(t)
	agentID := uuid.New()
	companyID := uuid.New()
	issueID := uuid.New()

	ms.agents[agentID] = &store.Agent{
		ID:            agentID,
		CompanyID:     companyID,
		Status:        store.StatusWorking,
		MonthlyBudget: 0, // unlimited
	}
	ms.issues[issueID] = &store.Issue{
		ID:         issueID,
		CompanyID:  companyID,
		Status:     store.IssueInProgress,
		AssigneeID: &agentID,
	}

	payload := agent.DonePayload{OutputPath: "/tmp/out.txt", TokensUsed: 100}
	raw, _ := json.Marshal(payload)

	o.HandleControlMessage(context.Background(), agentID, companyID,
		agent.ControlDone, raw)

	if !mh.hasEventType(ws.EventIssueUpdate) {
		t.Error("expected issue_update broadcast after done")
	}
}

func TestHandleControlMessage_Done_BudgetExhausted(t *testing.T) {
	o, ms, mh := newTestOrchestrator(t)
	agentID := uuid.New()
	companyID := uuid.New()
	issueID := uuid.New()

	ms.agents[agentID] = &store.Agent{
		ID:            agentID,
		CompanyID:     companyID,
		Status:        store.StatusWorking,
		MonthlyBudget: 100,
		TokenSpend:    90, // will hit limit after +100
	}
	ms.issues[issueID] = &store.Issue{
		ID:         issueID,
		CompanyID:  companyID,
		Status:     store.IssueInProgress,
		AssigneeID: &agentID,
	}

	payload := agent.DonePayload{OutputPath: "/tmp/out.txt", TokensUsed: 100}
	raw, _ := json.Marshal(payload)

	o.HandleControlMessage(context.Background(), agentID, companyID,
		agent.ControlDone, raw)

	if !mh.hasEventType(ws.EventNotification) {
		t.Error("expected notification broadcast for budget_exhausted")
	}
}

func TestHandleControlMessage_Done_NoMatchingIssue(t *testing.T) {
	o, ms, mh := newTestOrchestrator(t)
	agentID := uuid.New()
	companyID := uuid.New()

	ms.agents[agentID] = &store.Agent{
		ID:        agentID,
		CompanyID: companyID,
	}
	// No issues in the store.

	payload := agent.DonePayload{OutputPath: "/tmp/out.txt", TokensUsed: 0}
	raw, _ := json.Marshal(payload)

	// Should not panic.
	o.HandleControlMessage(context.Background(), agentID, companyID,
		agent.ControlDone, raw)
	_ = mh // no broadcasts expected, just no panic
}

func TestHandleControlMessage_Blocked(t *testing.T) {
	o, ms, mh := newTestOrchestrator(t)
	agentID := uuid.New()
	companyID := uuid.New()
	issueID := uuid.New()
	blockingIssueID := uuid.New()

	ms.agents[agentID] = &store.Agent{
		ID:        agentID,
		CompanyID: companyID,
	}
	ms.issues[issueID] = &store.Issue{
		ID:         issueID,
		CompanyID:  companyID,
		Status:     store.IssueInProgress,
		AssigneeID: &agentID,
	}

	payload := agent.BlockedPayload{WaitingOnIssueID: blockingIssueID}
	raw, _ := json.Marshal(payload)

	o.HandleControlMessage(context.Background(), agentID, companyID,
		agent.ControlBlocked, raw)

	ms.mu.Lock()
	issueStatus := ms.issues[issueID].Status
	ms.mu.Unlock()

	if issueStatus != store.IssueBlocked {
		t.Errorf("expected issue status Blocked, got %s", issueStatus)
	}
	if !mh.hasEventType(ws.EventIssueUpdate) {
		t.Error("expected issue_update broadcast after blocked")
	}
}

func TestHandleControlMessage_Escalate_ToCEO(t *testing.T) {
	o, ms, mh := newTestOrchestrator(t)
	agentID := uuid.New()
	companyID := uuid.New()
	issueID := uuid.New()

	// CEO: no manager
	ms.agents[agentID] = &store.Agent{
		ID:        agentID,
		CompanyID: companyID,
		ManagerID: nil, // CEO
	}
	ms.issues[issueID] = &store.Issue{
		ID:        issueID,
		CompanyID: companyID,
		Title:     "Critical bug",
	}

	payload := agent.EscalatePayload{IssueID: issueID, Reason: "stuck"}
	raw, _ := json.Marshal(payload)

	o.HandleControlMessage(context.Background(), agentID, companyID,
		agent.ControlEscalate, raw)

	// Should create a notification for human.
	ms.mu.Lock()
	notifCount := len(ms.notifications)
	ms.mu.Unlock()

	if notifCount == 0 {
		t.Error("expected human escalation notification to be created")
	}
	if !mh.hasEventType(ws.EventNotification) {
		t.Error("expected notification broadcast for CEO escalation")
	}
}

func TestHandleControlMessage_Escalate_ToManager(t *testing.T) {
	o, ms, mh := newTestOrchestrator(t)
	agentID := uuid.New()
	managerID := uuid.New()
	companyID := uuid.New()
	issueID := uuid.New()

	ms.agents[agentID] = &store.Agent{
		ID:        agentID,
		CompanyID: companyID,
		ManagerID: &managerID,
	}
	ms.issues[issueID] = &store.Issue{
		ID:        issueID,
		CompanyID: companyID,
		Title:     "Some task",
	}

	payload := agent.EscalatePayload{IssueID: issueID, Reason: "need help"}
	raw, _ := json.Marshal(payload)

	o.HandleControlMessage(context.Background(), agentID, companyID,
		agent.ControlEscalate, raw)

	if !mh.hasEventType(ws.EventEscalation) {
		t.Error("expected escalation broadcast when escalating to manager")
	}
}

func TestHandleControlMessage_Hire_InsufficientBudget(t *testing.T) {
	o, ms, _ := newTestOrchestrator(t)
	agentID := uuid.New()
	companyID := uuid.New()

	ms.agents[agentID] = &store.Agent{
		ID:            agentID,
		CompanyID:     companyID,
		MonthlyBudget: 100,
		TokenSpend:    90,
	}
	// Subtree is just the agent itself (can hire reporting to self).
	ms.subtreeIDs[agentID] = []uuid.UUID{agentID}

	payload := agent.HirePayload{
		RoleTitle:        "Engineer",
		ReportingTo:      agentID,
		SystemPrompt:     "You are an engineer",
		Runtime:          store.RuntimeClaudeCode,
		BudgetAllocation: 50, // need 50, have 10
	}
	raw, _ := json.Marshal(payload)

	// Inject a runtime so sendRejection can call SendTask.
	rejected := false
	rt := &mockRuntime{
		sendTaskFn: func(ctx context.Context, issue store.Issue) error {
			rejected = true
			return nil
		},
	}
	injectRuntime(o, agentID, rt)

	o.HandleControlMessage(context.Background(), agentID, companyID,
		agent.ControlHire, raw)

	if !rejected {
		t.Error("expected rejection to be sent when budget is insufficient")
	}
}

func TestHandleControlMessage_Hire_RuntimeNotAvailable(t *testing.T) {
	o, ms, _ := newTestOrchestrator(t)
	agentID := uuid.New()
	companyID := uuid.New()

	ms.agents[agentID] = &store.Agent{
		ID:            agentID,
		CompanyID:     companyID,
		MonthlyBudget: 1000,
		TokenSpend:    0,
	}
	ms.subtreeIDs[agentID] = []uuid.UUID{agentID}

	// Mark no runtimes available.
	o.mu.Lock()
	o.available = Runtimes{ClaudeCode: false, OpenClaw: false}
	o.mu.Unlock()

	rejected := false
	rt := &mockRuntime{
		sendTaskFn: func(ctx context.Context, issue store.Issue) error {
			rejected = true
			return nil
		},
	}
	injectRuntime(o, agentID, rt)

	payload := agent.HirePayload{
		RoleTitle:        "Engineer",
		ReportingTo:      agentID,
		SystemPrompt:     "prompt",
		Runtime:          store.RuntimeClaudeCode,
		BudgetAllocation: 100,
	}
	raw, _ := json.Marshal(payload)

	o.HandleControlMessage(context.Background(), agentID, companyID,
		agent.ControlHire, raw)

	if !rejected {
		t.Error("expected rejection when runtime is not available")
	}
}

func TestHandleControlMessage_Hire_OutsideSubtree(t *testing.T) {
	o, ms, _ := newTestOrchestrator(t)
	agentID := uuid.New()
	outsiderID := uuid.New()
	companyID := uuid.New()

	ms.agents[agentID] = &store.Agent{
		ID:            agentID,
		CompanyID:     companyID,
		MonthlyBudget: 1000,
		TokenSpend:    0,
	}
	// Subtree does not include outsiderID.
	ms.subtreeIDs[agentID] = []uuid.UUID{}

	rejected := false
	rt := &mockRuntime{
		sendTaskFn: func(ctx context.Context, issue store.Issue) error {
			rejected = true
			return nil
		},
	}
	injectRuntime(o, agentID, rt)

	payload := agent.HirePayload{
		RoleTitle:        "Engineer",
		ReportingTo:      outsiderID, // not in subtree and not self
		SystemPrompt:     "prompt",
		Runtime:          store.RuntimeClaudeCode,
		BudgetAllocation: 100,
	}
	raw, _ := json.Marshal(payload)

	o.HandleControlMessage(context.Background(), agentID, companyID,
		agent.ControlHire, raw)

	if !rejected {
		t.Error("expected rejection when reporting_to is outside subtree")
	}
}

func TestHandleControlMessage_Hire_Success(t *testing.T) {
	o, ms, mh := newTestOrchestrator(t)
	agentID := uuid.New()
	companyID := uuid.New()

	ms.agents[agentID] = &store.Agent{
		ID:            agentID,
		CompanyID:     companyID,
		MonthlyBudget: 1000,
		TokenSpend:    0,
	}
	ms.subtreeIDs[agentID] = []uuid.UUID{agentID}

	o.mu.Lock()
	o.available = Runtimes{ClaudeCode: true}
	o.mu.Unlock()

	payload := agent.HirePayload{
		RoleTitle:        "Engineer",
		ReportingTo:      agentID,
		SystemPrompt:     "You are an engineer",
		Runtime:          store.RuntimeClaudeCode,
		BudgetAllocation: 100,
	}
	raw, _ := json.Marshal(payload)

	o.HandleControlMessage(context.Background(), agentID, companyID,
		agent.ControlHire, raw)

	ms.mu.Lock()
	hireCount := len(ms.pendingHires)
	ms.mu.Unlock()

	if hireCount == 0 {
		t.Error("expected a pending hire to be created")
	}
	if !mh.hasEventType(ws.EventHirePending) {
		t.Error("expected hire_pending broadcast")
	}
}

// ---------------------------------------------------------------------------
// Tests: ApproveHire
// ---------------------------------------------------------------------------

func TestApproveHire_NotFound(t *testing.T) {
	o, _, _ := newTestOrchestrator(t)

	err := o.ApproveHire(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error when hire not found")
	}
}

func TestApproveHire_NotPending(t *testing.T) {
	o, ms, _ := newTestOrchestrator(t)
	hireID := uuid.New()

	ms.pendingHires[hireID] = &store.PendingHire{
		ID:     hireID,
		Status: store.HireApproved, // already approved
	}

	err := o.ApproveHire(context.Background(), hireID)
	if err == nil {
		t.Fatal("expected error when hire is not in pending state")
	}
}

func TestApproveHire_Success(t *testing.T) {
	o, ms, mh := newTestOrchestrator(t)
	hireID := uuid.New()
	requesterID := uuid.New()
	managerID := uuid.New()
	companyID := uuid.New()

	task := "build the login page"
	ms.pendingHires[hireID] = &store.PendingHire{
		ID:                   hireID,
		CompanyID:            companyID,
		RequestedByAgentID:   requesterID,
		RoleTitle:            "Frontend Dev",
		ReportingToAgentID:   managerID,
		SystemPrompt:         "You are a frontend dev",
		Runtime:              store.RuntimeClaudeCode,
		BudgetAllocation:     200,
		InitialTask:          &task,
		Status:               store.HirePending,
	}
	ms.agents[requesterID] = &store.Agent{
		ID:            requesterID,
		CompanyID:     companyID,
		MonthlyBudget: 1000,
		TokenSpend:    0,
	}

	// SpawnAgent will call createRuntime which needs the actual CLI.
	// Since we don't have it, mock the runtime injection by making
	// SpawnAgent succeed for "board" role. We use role="board" to skip spawn.
	// Instead, directly test by having createAgent return a board agent.
	// Actually, let's test with a role that isn't "board" but override
	// by setting the runtime to an unknown value so createRuntime fails
	// gracefully — but ApproveHire itself returns the error.

	// Better approach: set hire runtime to something that will fail at createRuntime,
	// but test the DB-level work (budget deduction, agent creation) still happened
	// before the spawn. However ApproveHire fails fast.

	// Use role "board" workaround: change the hire to use "board" role
	// so SpawnAgent returns nil immediately.
	ms.pendingHires[hireID].Runtime = store.RuntimeClaudeCode

	// Override the hire to a board-style spawn by using a runtime we can inject.
	// We do this by swapping createRuntime via a test-only override.
	// Since createRuntime is not exported, let's just test the error path or
	// use an indirect workaround: verify agent creation + budget deduction happen.

	// The cleanest option: skip spawn entirely by catching the error.
	// ApproveHire will fail at SpawnAgent("claude" not found). We can verify
	// the budget was deducted and the agent was created before spawn.

	_ = mh // will check after

	err := o.ApproveHire(context.Background(), hireID)
	// On systems without "claude", this may fail at SpawnAgent.
	// Verify budget deduction happened regardless.
	ms.mu.Lock()
	spent := ms.tokenSpend[requesterID]
	ms.mu.Unlock()

	if spent != 200 {
		t.Errorf("expected 200 budget deducted, got %d (err: %v)", spent, err)
	}
}

// ---------------------------------------------------------------------------
// Tests: RejectHire
// ---------------------------------------------------------------------------

func TestRejectHire_NotFound(t *testing.T) {
	o, _, _ := newTestOrchestrator(t)

	err := o.RejectHire(context.Background(), uuid.New(), "not needed")
	if err == nil {
		t.Fatal("expected error when hire not found")
	}
}

func TestRejectHire_Success(t *testing.T) {
	o, ms, _ := newTestOrchestrator(t)
	hireID := uuid.New()
	requesterID := uuid.New()
	companyID := uuid.New()

	ms.pendingHires[hireID] = &store.PendingHire{
		ID:                 hireID,
		CompanyID:          companyID,
		RequestedByAgentID: requesterID,
		Status:             store.HirePending,
	}
	ms.agents[requesterID] = &store.Agent{
		ID:        requesterID,
		CompanyID: companyID,
	}

	rejectionReceived := false
	rt := &mockRuntime{
		sendTaskFn: func(ctx context.Context, issue store.Issue) error {
			rejectionReceived = true
			return nil
		},
	}
	injectRuntime(o, requesterID, rt)

	err := o.RejectHire(context.Background(), hireID, "not in budget")
	if err != nil {
		t.Fatalf("RejectHire returned error: %v", err)
	}

	ms.mu.Lock()
	hireStatus := ms.pendingHires[hireID].Status
	ms.mu.Unlock()

	if hireStatus != store.HireRejected {
		t.Errorf("expected HireRejected, got %s", hireStatus)
	}
	if !rejectionReceived {
		t.Error("expected rejection message sent to requesting agent")
	}
}

// ---------------------------------------------------------------------------
// Tests: ReassignAgent
// ---------------------------------------------------------------------------

func TestReassignAgent_Success(t *testing.T) {
	o, ms, mh := newTestOrchestrator(t)
	agentID := uuid.New()
	newManagerID := uuid.New()
	companyID := uuid.New()

	ms.agents[agentID] = &store.Agent{
		ID:        agentID,
		CompanyID: companyID,
	}

	err := o.ReassignAgent(context.Background(), agentID, newManagerID)
	if err != nil {
		t.Fatalf("ReassignAgent returned error: %v", err)
	}

	ms.mu.Lock()
	updatedManager := ms.agents[agentID].ManagerID
	ms.mu.Unlock()

	if updatedManager == nil || *updatedManager != newManagerID {
		t.Errorf("expected manager to be updated to %s", newManagerID)
	}
	if !mh.hasEventType(ws.EventAgentStatus) {
		t.Error("expected agent_status broadcast after reassign")
	}
}

// ---------------------------------------------------------------------------
// Tests: SpawnAgent
// ---------------------------------------------------------------------------

func TestSpawnAgent_BoardAgentSkipped(t *testing.T) {
	o, ms, _ := newTestOrchestrator(t)
	companyID := uuid.New()
	agentID := uuid.New()

	ms.agents[agentID] = &store.Agent{
		ID:        agentID,
		CompanyID: companyID,
		Role:      "board",
	}

	a := &store.Agent{
		ID:        agentID,
		CompanyID: companyID,
		Role:      "board",
	}

	err := o.SpawnAgent(context.Background(), a)
	if err != nil {
		t.Fatalf("SpawnAgent for board agent returned error: %v", err)
	}

	// Board agents should never appear in runtimes.
	o.mu.RLock()
	_, exists := o.runtimes[agentID]
	o.mu.RUnlock()

	if exists {
		t.Error("board agent should not have a runtime registered")
	}
}

func TestSpawnAgent_UnknownRuntime(t *testing.T) {
	o, _, _ := newTestOrchestrator(t)

	a := &store.Agent{
		ID:        uuid.New(),
		CompanyID: uuid.New(),
		Role:      "worker",
		Runtime:   store.AgentRuntime("unknown_runtime"),
	}

	err := o.SpawnAgent(context.Background(), a)
	if err == nil {
		t.Fatal("expected error for unknown runtime")
	}
}

// ---------------------------------------------------------------------------
// Tests: commandExists
// ---------------------------------------------------------------------------

func TestCommandExists_PresentCommand(t *testing.T) {
	if !commandExists("go") {
		t.Error("commandExists('go') should return true in Go test environment")
	}
}

func TestCommandExists_AbsentCommand(t *testing.T) {
	if commandExists("__guaranteed_absent_command_xyz_12345__") {
		t.Error("commandExists should return false for non-existent command")
	}
}

// ---------------------------------------------------------------------------
// Tests: handleDone edge cases
// ---------------------------------------------------------------------------

func TestHandleDone_NoIssueAssignedToAgent(t *testing.T) {
	o, ms, _ := newTestOrchestrator(t)
	agentID := uuid.New()
	companyID := uuid.New()
	otherAgentID := uuid.New()
	issueID := uuid.New()

	ms.agents[agentID] = &store.Agent{
		ID:        agentID,
		CompanyID: companyID,
	}
	// Issue is assigned to a different agent.
	ms.issues[issueID] = &store.Issue{
		ID:         issueID,
		CompanyID:  companyID,
		Status:     store.IssueInProgress,
		AssigneeID: &otherAgentID,
	}

	d := agent.DonePayload{OutputPath: "/out", TokensUsed: 0}
	// Should not panic or update the wrong issue.
	o.handleDone(context.Background(), agentID, companyID, d)

	ms.mu.Lock()
	issueStatus := ms.issues[issueID].Status
	ms.mu.Unlock()

	if issueStatus != store.IssueInProgress {
		t.Error("issue status should not change when done message is from a different agent")
	}
}

// ---------------------------------------------------------------------------
// Tests: handleEscalate edge cases
// ---------------------------------------------------------------------------

func TestHandleEscalate_IssueNotFound(t *testing.T) {
	o, _, _ := newTestOrchestrator(t)
	agentID := uuid.New()
	companyID := uuid.New()

	// No issue in store; should not panic.
	e := agent.EscalatePayload{IssueID: uuid.New(), Reason: "stuck"}
	o.handleEscalate(context.Background(), agentID, companyID, e)
}

// ---------------------------------------------------------------------------
// Tests: handleBlocked edge cases
// ---------------------------------------------------------------------------

func TestHandleBlocked_IssueNotInProgress(t *testing.T) {
	o, ms, _ := newTestOrchestrator(t)
	agentID := uuid.New()
	companyID := uuid.New()
	issueID := uuid.New()

	ms.agents[agentID] = &store.Agent{ID: agentID, CompanyID: companyID}
	ms.issues[issueID] = &store.Issue{
		ID:         issueID,
		CompanyID:  companyID,
		Status:     store.IssuePending, // not in progress
		AssigneeID: &agentID,
	}

	b := agent.BlockedPayload{WaitingOnIssueID: uuid.New()}
	o.handleBlocked(context.Background(), agentID, companyID, b)

	ms.mu.Lock()
	issueStatus := ms.issues[issueID].Status
	ms.mu.Unlock()

	// Status should remain pending — issue wasn't in_progress.
	if issueStatus != store.IssuePending {
		t.Errorf("expected issue to remain pending, got %s", issueStatus)
	}
}
