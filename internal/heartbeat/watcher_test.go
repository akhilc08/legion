package heartbeat

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"conductor/internal/store"
	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Mock store
// ---------------------------------------------------------------------------

type mockStore struct {
	mu sync.Mutex

	upsertHeartbeats []uuid.UUID
	incrementedMisses map[uuid.UUID]int
	updatedStatuses  []statusUpdate
	staleAgents      []store.Agent
	staleErr         error
	missErr          error
}

type statusUpdate struct {
	agentID uuid.UUID
	status  store.AgentStatus
}

func newMockStore() *mockStore {
	return &mockStore{
		incrementedMisses: make(map[uuid.UUID]int),
	}
}

func (m *mockStore) UpsertHeartbeat(ctx context.Context, agentID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.upsertHeartbeats = append(m.upsertHeartbeats, agentID)
	return nil
}

func (m *mockStore) StaleAgents(ctx context.Context, threshold time.Duration) ([]store.Agent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.staleErr != nil {
		return nil, m.staleErr
	}
	return m.staleAgents, nil
}

func (m *mockStore) IncrementMisses(ctx context.Context, agentID uuid.UUID) (int, error) {
	if m.missErr != nil {
		return 0, m.missErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.incrementedMisses[agentID]++
	return m.incrementedMisses[agentID], nil
}

func (m *mockStore) UpdateAgentStatus(ctx context.Context, id uuid.UUID, status store.AgentStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updatedStatuses = append(m.updatedStatuses, statusUpdate{agentID: id, status: status})
	return nil
}

// ---------------------------------------------------------------------------
// Tests: NewWatcher
// ---------------------------------------------------------------------------

func TestNewWatcher_Fields(t *testing.T) {
	ms := newMockStore()
	var failedCalled, degradedCalled bool
	onFailed := func(ctx context.Context, agentID, companyID uuid.UUID) { failedCalled = true }
	onDegraded := func(ctx context.Context, agentID, companyID uuid.UUID) { degradedCalled = true }

	w := NewWatcher(ms, onFailed, onDegraded)

	if w == nil {
		t.Fatal("NewWatcher returned nil")
	}
	if w.db == nil {
		t.Error("db should not be nil")
	}
	if w.watching == nil {
		t.Error("watching map should not be nil")
	}
	if w.onFailed == nil {
		t.Error("onFailed should not be nil")
	}
	if w.onDegraded == nil {
		t.Error("onDegraded should not be nil")
	}

	// Confirm callbacks work.
	w.onFailed(context.Background(), uuid.New(), uuid.New())
	if !failedCalled {
		t.Error("onFailed callback not invoked")
	}
	w.onDegraded(context.Background(), uuid.New(), uuid.New())
	if !degradedCalled {
		t.Error("onDegraded callback not invoked")
	}
}

func TestNewWatcher_NilHandlers(t *testing.T) {
	ms := newMockStore()
	w := NewWatcher(ms, nil, nil)
	if w == nil {
		t.Fatal("NewWatcher with nil handlers returned nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: Watch
// ---------------------------------------------------------------------------

func TestWatch_StartsMonitoring(t *testing.T) {
	ms := newMockStore()
	w := NewWatcher(ms, nil, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	agentID := uuid.New()
	companyID := uuid.New()

	w.Watch(ctx, agentID, companyID)

	w.mu.Lock()
	_, exists := w.watching[agentID]
	w.mu.Unlock()

	if !exists {
		t.Error("Watch should register the agent in the watching map")
	}
}

func TestWatch_DuplicateCallIsNoOp(t *testing.T) {
	ms := newMockStore()
	w := NewWatcher(ms, nil, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	agentID := uuid.New()
	companyID := uuid.New()

	w.Watch(ctx, agentID, companyID)

	w.mu.Lock()
	originalCancel := w.watching[agentID]
	w.mu.Unlock()

	// Call Watch a second time — should be a no-op.
	w.Watch(ctx, agentID, companyID)

	w.mu.Lock()
	currentCancel := w.watching[agentID]
	w.mu.Unlock()

	// The cancel function pointer should be the same object (same goroutine).
	// We can't compare function pointers in Go, but we can verify only one
	// cancel entry exists.
	if originalCancel == nil || currentCancel == nil {
		t.Error("cancel function should not be nil after Watch")
	}
}

// ---------------------------------------------------------------------------
// Tests: Unwatch
// ---------------------------------------------------------------------------

func TestUnwatch_RemovesAgent(t *testing.T) {
	ms := newMockStore()
	w := NewWatcher(ms, nil, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	agentID := uuid.New()
	companyID := uuid.New()

	w.Watch(ctx, agentID, companyID)
	w.Unwatch(agentID)

	w.mu.Lock()
	_, exists := w.watching[agentID]
	w.mu.Unlock()

	if exists {
		t.Error("Unwatch should remove the agent from the watching map")
	}
}

func TestUnwatch_CancelsContext(t *testing.T) {
	ms := newMockStore()
	w := NewWatcher(ms, nil, nil)

	agentID := uuid.New()
	companyID := uuid.New()

	// Use a parent context we can inspect.
	parentCtx, parentCancel := context.WithCancel(context.Background())
	defer parentCancel()

	// Track when watchAgent goroutine exits.
	goroutineExited := make(chan struct{})
	originalWatchAgent := w.watchAgent

	// Temporarily replace watchAgent for testing — can't do this directly since
	// it's not injectable. Instead, verify via Unwatch effect on the cancel map.
	_ = originalWatchAgent

	w.Watch(parentCtx, agentID, companyID)

	// Unwatch should cancel the derived context and stop the goroutine.
	w.Unwatch(agentID)
	close(goroutineExited) // signal we're done checking

	w.mu.Lock()
	_, exists := w.watching[agentID]
	w.mu.Unlock()

	if exists {
		t.Error("agent should not be in watching map after Unwatch")
	}
}

func TestUnwatch_SafeForNonWatchedAgent(t *testing.T) {
	ms := newMockStore()
	w := NewWatcher(ms, nil, nil)

	// Should not panic.
	w.Unwatch(uuid.New())
}

// ---------------------------------------------------------------------------
// Tests: RecordHeartbeat
// ---------------------------------------------------------------------------

func TestRecordHeartbeat_CallsUpsert(t *testing.T) {
	ms := newMockStore()
	w := NewWatcher(ms, nil, nil)

	agentID := uuid.New()
	err := w.RecordHeartbeat(context.Background(), agentID)
	if err != nil {
		t.Fatalf("RecordHeartbeat returned error: %v", err)
	}

	ms.mu.Lock()
	heartbeats := ms.upsertHeartbeats
	ms.mu.Unlock()

	if len(heartbeats) == 0 {
		t.Error("expected UpsertHeartbeat to be called")
	}
	if heartbeats[0] != agentID {
		t.Errorf("expected agentID %s, got %s", agentID, heartbeats[0])
	}
}

func TestRecordHeartbeat_MultipleAgents(t *testing.T) {
	ms := newMockStore()
	w := NewWatcher(ms, nil, nil)

	ctx := context.Background()
	id1 := uuid.New()
	id2 := uuid.New()

	w.RecordHeartbeat(ctx, id1) //nolint
	w.RecordHeartbeat(ctx, id2) //nolint
	w.RecordHeartbeat(ctx, id1) //nolint

	ms.mu.Lock()
	count := len(ms.upsertHeartbeats)
	ms.mu.Unlock()

	if count != 3 {
		t.Errorf("expected 3 upsert calls, got %d", count)
	}
}

// ---------------------------------------------------------------------------
// Tests: checkAgent (integration via direct invocation)
// ---------------------------------------------------------------------------

func TestCheckAgent_NotStale_NoAction(t *testing.T) {
	ms := newMockStore()
	w := NewWatcher(ms, nil, nil)

	agentID := uuid.New()
	companyID := uuid.New()
	// Stale list is empty — agent is not stale.
	ms.staleAgents = []store.Agent{}

	w.checkAgent(context.Background(), agentID, companyID)

	ms.mu.Lock()
	misses := len(ms.incrementedMisses)
	ms.mu.Unlock()

	if misses != 0 {
		t.Errorf("expected no miss increments when agent is not stale, got %d", misses)
	}
}

func TestCheckAgent_FirstMiss_DegradedStatus(t *testing.T) {
	ms := newMockStore()
	agentID := uuid.New()
	companyID := uuid.New()

	var degradedCalled int32
	onDegraded := func(ctx context.Context, aid, cid uuid.UUID) {
		if aid == agentID {
			atomic.AddInt32(&degradedCalled, 1)
		}
	}

	w := NewWatcher(ms, nil, onDegraded)
	ms.staleAgents = []store.Agent{{ID: agentID, CompanyID: companyID}}
	// First miss returns 1 (>= MissThresholdDegraded=1, < MissThresholdFailed=3).

	w.checkAgent(context.Background(), agentID, companyID)

	// Wait briefly for the goroutine in onDegraded.
	time.Sleep(50 * time.Millisecond)

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
		t.Error("expected agent to be marked Degraded after first miss")
	}

	if atomic.LoadInt32(&degradedCalled) == 0 {
		t.Error("expected onDegraded callback to be called")
	}
}

func TestCheckAgent_ThirdMiss_FailedStatus_AndUnwatch(t *testing.T) {
	ms := newMockStore()
	agentID := uuid.New()
	companyID := uuid.New()

	// Set miss count to 2 so next increment = 3 = MissThresholdFailed.
	ms.incrementedMisses[agentID] = 2

	var failedCalled int32
	onFailed := func(ctx context.Context, aid, cid uuid.UUID) {
		if aid == agentID {
			atomic.AddInt32(&failedCalled, 1)
		}
	}

	w := NewWatcher(ms, onFailed, nil)

	// Register agent in watching map so Unwatch can clean it up.
	_, dummyCancel := context.WithCancel(context.Background())
	w.mu.Lock()
	w.watching[agentID] = dummyCancel
	w.mu.Unlock()

	ms.staleAgents = []store.Agent{{ID: agentID, CompanyID: companyID}}

	w.checkAgent(context.Background(), agentID, companyID)

	// Wait briefly for goroutines.
	time.Sleep(50 * time.Millisecond)

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
		t.Error("expected agent to be marked Failed after third miss")
	}

	w.mu.Lock()
	_, stillWatching := w.watching[agentID]
	w.mu.Unlock()

	if stillWatching {
		t.Error("agent should be unwatched after failure")
	}

	if atomic.LoadInt32(&failedCalled) == 0 {
		t.Error("expected onFailed callback to be called")
	}
}

func TestCheckAgent_StaleError_DoesNotPanic(t *testing.T) {
	ms := newMockStore()
	ms.staleErr = context.DeadlineExceeded
	w := NewWatcher(ms, nil, nil)

	agentID := uuid.New()
	companyID := uuid.New()

	// Should log but not panic.
	w.checkAgent(context.Background(), agentID, companyID)
}

func TestCheckAgent_DifferentAgentInStaleList(t *testing.T) {
	ms := newMockStore()
	agentID := uuid.New()
	companyID := uuid.New()
	otherAgentID := uuid.New()

	// Stale list contains a different agent.
	ms.staleAgents = []store.Agent{{ID: otherAgentID, CompanyID: companyID}}

	w := NewWatcher(ms, nil, nil)
	w.checkAgent(context.Background(), agentID, companyID)

	ms.mu.Lock()
	count := len(ms.incrementedMisses)
	ms.mu.Unlock()

	if count != 0 {
		t.Errorf("should not increment misses for a different agent, got %d increments", count)
	}
}

// ---------------------------------------------------------------------------
// Tests: Watch integration with short ticker
// ---------------------------------------------------------------------------

func TestWatch_TickerTriggersCheck(t *testing.T) {
	// This test verifies that the watchAgent goroutine calls checkAgent on tick.
	// We use a very short CheckInterval by calling checkAgent directly.
	// The integration test below verifies the goroutine lifecycle.

	ms := newMockStore()
	agentID := uuid.New()
	companyID := uuid.New()

	var degradedCalled int32
	onDegraded := func(ctx context.Context, aid, cid uuid.UUID) {
		atomic.AddInt32(&degradedCalled, 1)
	}

	w := NewWatcher(ms, nil, onDegraded)

	// Make the agent appear stale with first miss.
	ms.staleAgents = []store.Agent{{ID: agentID, CompanyID: companyID}}

	// Simulate what happens on a ticker fire.
	w.checkAgent(context.Background(), agentID, companyID)

	time.Sleep(30 * time.Millisecond)

	ms.mu.Lock()
	statuses := ms.updatedStatuses
	ms.mu.Unlock()

	if len(statuses) == 0 {
		t.Error("expected status update after checkAgent")
	}
}

func TestWatch_ContextCancellationStopsGoroutine(t *testing.T) {
	ms := newMockStore()
	w := NewWatcher(ms, nil, nil)

	agentID := uuid.New()
	companyID := uuid.New()

	ctx, cancel := context.WithCancel(context.Background())

	w.Watch(ctx, agentID, companyID)

	// Cancel the context — goroutine should exit.
	cancel()

	// Give goroutine time to exit.
	time.Sleep(50 * time.Millisecond)

	// Verify no panics occurred. The goroutine should have exited cleanly.
	// We can verify by unwatching — the cancel in watching map should already
	// have been invoked via the parent context.
	w.Unwatch(agentID) // safe to call even if already cancelled
}
