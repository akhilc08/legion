package orchestrator

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"conductor/internal/store"
	"conductor/internal/ws"
	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Minimal mocks for benchmarks
// ---------------------------------------------------------------------------

type benchStore struct {
	mu       sync.Mutex
	issues   []store.Issue
	checkouts int64
}

func (s *benchStore) ListCompanies(ctx context.Context) ([]store.Company, error) { return nil, nil }
func (s *benchStore) ListAgentsByCompany(ctx context.Context, cid uuid.UUID) ([]store.Agent, error) {
	return nil, nil
}
func (s *benchStore) CreateAgent(ctx context.Context, a *store.Agent) (*store.Agent, error) {
	a.ID = uuid.New()
	return a, nil
}
func (s *benchStore) GetAgent(ctx context.Context, id uuid.UUID) (*store.Agent, error) {
	return &store.Agent{ID: id}, nil
}
func (s *benchStore) DeleteAgent(ctx context.Context, id uuid.UUID) error     { return nil }
func (s *benchStore) UpdateAgentStatus(_ context.Context, _ uuid.UUID, _ store.AgentStatus) error {
	return nil
}
func (s *benchStore) UpdateAgentPID(_ context.Context, _ uuid.UUID, _ *int) error  { return nil }
func (s *benchStore) UpdateAgentManager(_ context.Context, _, _ uuid.UUID) error   { return nil }
func (s *benchStore) AddTokenSpend(_ context.Context, _ uuid.UUID, _ int, _ bool) error {
	return nil
}
func (s *benchStore) GetSubtreeAgentIDs(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}
func (s *benchStore) ListIssuesByCompany(_ context.Context, _ uuid.UUID) ([]store.Issue, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := make([]store.Issue, len(s.issues))
	copy(cp, s.issues)
	return cp, nil
}
func (s *benchStore) CreateIssue(_ context.Context, issue *store.Issue) (*store.Issue, error) {
	issue.ID = uuid.New()
	s.mu.Lock()
	s.issues = append(s.issues, *issue)
	s.mu.Unlock()
	return issue, nil
}
func (s *benchStore) GetIssue(_ context.Context, id uuid.UUID) (*store.Issue, error) {
	return &store.Issue{ID: id}, nil
}
func (s *benchStore) UpdateIssueAssignee(_ context.Context, _, _ uuid.UUID) error  { return nil }
func (s *benchStore) UpdateIssueStatus(_ context.Context, _ uuid.UUID, _ store.IssueStatus) error {
	return nil
}
func (s *benchStore) UpdateIssueOutput(_ context.Context, _ uuid.UUID, _ string) error { return nil }
func (s *benchStore) IncrementAttemptCount(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}
func (s *benchStore) CheckoutIssue(_ context.Context, _ uuid.UUID, _ uuid.UUID) (bool, error) {
	atomic.AddInt64(&s.checkouts, 1)
	return true, nil
}
func (s *benchStore) ListReadyIssues(_ context.Context, _ uuid.UUID) ([]store.Issue, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := make([]store.Issue, len(s.issues))
	copy(cp, s.issues)
	return cp, nil
}
func (s *benchStore) AddDependency(_ context.Context, _, _ uuid.UUID) error { return nil }
func (s *benchStore) CreatePendingHire(_ context.Context, h *store.PendingHire) (*store.PendingHire, error) {
	h.ID = uuid.New()
	return h, nil
}
func (s *benchStore) GetPendingHire(_ context.Context, id uuid.UUID) (*store.PendingHire, error) {
	return &store.PendingHire{ID: id}, nil
}
func (s *benchStore) UpdateHireStatus(_ context.Context, _ uuid.UUID, _ store.HireStatus) error {
	return nil
}
func (s *benchStore) UpsertHeartbeat(_ context.Context, _ uuid.UUID) error { return nil }
func (s *benchStore) IncrementMisses(_ context.Context, _ uuid.UUID) (int, error) { return 0, nil }
func (s *benchStore) StaleAgents(_ context.Context, _ time.Duration) ([]store.Agent, error) {
	return nil, nil
}
func (s *benchStore) CascadePermissionsFromManager(_ context.Context, _, _ uuid.UUID) error {
	return nil
}
func (s *benchStore) CreateNotification(_ context.Context, n *store.Notification) (*store.Notification, error) {
	n.ID = uuid.New()
	return n, nil
}
func (s *benchStore) Log(_ context.Context, _ uuid.UUID, _ *uuid.UUID, _ string, _ map[string]interface{}) error {
	return nil
}

type benchHub struct{}

func (h *benchHub) Broadcast(_ ws.Event)    {}
func (h *benchHub) BroadcastAll(_ ws.Event) {}

// ---------------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------------

// BenchmarkCheckoutIssue measures advisory-lock-equivalent checkout contention.
// In production this maps to a Postgres SELECT ... FOR UPDATE advisory lock.
// Here we measure pure coordinator overhead without DB latency.
func BenchmarkCheckoutIssue_Sequential(b *testing.B) {
	db := &benchStore{}
	ctx := context.Background()
	agentID := uuid.New()
	companyID := uuid.New()

	// Pre-populate issues
	for i := 0; i < 100; i++ {
		_, _ = db.CreateIssue(ctx, &store.Issue{
			CompanyID: companyID,
			Status:    store.IssuePending,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = db.CheckoutIssue(ctx, db.issues[i%len(db.issues)].ID, agentID)
	}
}

// BenchmarkCheckoutIssue_Concurrent simulates N agents competing for issues simultaneously.
func BenchmarkCheckoutIssue_Concurrent_10Workers(b *testing.B) {
	benchmarkConcurrentCheckout(b, 10)
}

func BenchmarkCheckoutIssue_Concurrent_50Workers(b *testing.B) {
	benchmarkConcurrentCheckout(b, 50)
}

func benchmarkConcurrentCheckout(b *testing.B, workers int) {
	db := &benchStore{}
	ctx := context.Background()
	companyID := uuid.New()

	for i := 0; i < 1000; i++ {
		_, _ = db.CreateIssue(ctx, &store.Issue{
			CompanyID: companyID,
			Status:    store.IssuePending,
		})
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		agentID := uuid.New()
		i := 0
		for pb.Next() {
			idx := i % len(db.issues)
			_, _ = db.CheckoutIssue(ctx, db.issues[idx].ID, agentID)
			i++
		}
	})
}

// BenchmarkOrchestratorWakeChannel measures the wake signal dispatch overhead
// used to notify an agent's assign loop to check for new issues immediately.
func BenchmarkOrchestratorWakeChannel(b *testing.B) {
	const numAgents = 50
	wakeChans := make(map[uuid.UUID]chan struct{}, numAgents)
	ids := make([]uuid.UUID, numAgents)
	for i := range ids {
		ids[i] = uuid.New()
		wakeChans[ids[i]] = make(chan struct{}, 1)
	}

	// Drain goroutines
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for _, ch := range wakeChans {
		ch := ch
		go func() {
			for {
				select {
				case <-ch:
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ch := wakeChans[ids[i%numAgents]]
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

// BenchmarkOrchestratorLockContention measures RWMutex overhead on the runtime map
// under concurrent read (status checks) and occasional write (agent spawn/kill).
func BenchmarkOrchestratorLockContention_ReadHeavy(b *testing.B) {
	var mu sync.RWMutex
	runtimes := make(map[uuid.UUID]struct{})
	for i := 0; i < 20; i++ {
		runtimes[uuid.New()] = struct{}{}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.RLock()
			_ = len(runtimes)
			mu.RUnlock()
		}
	})
}

func BenchmarkOrchestratorLockContention_WriteContention(b *testing.B) {
	var mu sync.RWMutex
	runtimes := make(map[uuid.UUID]struct{})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		id := uuid.New()
		i := 0
		for pb.Next() {
			if i%10 == 0 {
				// 10% writes (agent spawn/kill)
				mu.Lock()
				runtimes[id] = struct{}{}
				mu.Unlock()
			} else {
				// 90% reads (status checks, issue assignment)
				mu.RLock()
				_ = len(runtimes)
				mu.RUnlock()
			}
			i++
		}
	})
}
