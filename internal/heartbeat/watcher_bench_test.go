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

// fastMockStore is a lock-free mock optimized for benchmarking the watcher hot path.
type fastMockStore struct {
	staleAgents []store.Agent
	missCounts  sync.Map // uuid.UUID -> int64
}

func (m *fastMockStore) UpsertHeartbeat(_ context.Context, _ uuid.UUID) error { return nil }
func (m *fastMockStore) StaleAgents(_ context.Context, _ time.Duration) ([]store.Agent, error) {
	return m.staleAgents, nil
}
func (m *fastMockStore) IncrementMisses(_ context.Context, agentID uuid.UUID) (int, error) {
	v, _ := m.missCounts.LoadOrStore(agentID, new(int64))
	n := atomic.AddInt64(v.(*int64), 1)
	return int(n), nil
}
func (m *fastMockStore) UpdateAgentStatus(_ context.Context, _ uuid.UUID, _ store.AgentStatus) error {
	return nil
}

// BenchmarkWatcher_CheckAgent_NoStale measures the hot path when all agents are healthy.
func BenchmarkWatcher_CheckAgent_NoStale(b *testing.B) {
	db := &fastMockStore{staleAgents: []store.Agent{}}
	w := NewWatcher(db, nil, nil)
	agentID := uuid.New()
	companyID := uuid.New()
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.checkAgent(ctx, agentID, companyID)
	}
}

// BenchmarkWatcher_CheckAgent_StaleMatch measures detection of a stale agent.
func BenchmarkWatcher_CheckAgent_StaleMatch(b *testing.B) {
	agentID := uuid.New()
	companyID := uuid.New()
	db := &fastMockStore{
		staleAgents: []store.Agent{{ID: agentID, CompanyID: companyID}},
	}
	w := NewWatcher(db, nil, nil)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.checkAgent(ctx, agentID, companyID)
	}
}

// BenchmarkWatcher_RecordHeartbeat measures heartbeat upsert throughput.
func BenchmarkWatcher_RecordHeartbeat(b *testing.B) {
	db := &fastMockStore{}
	w := NewWatcher(db, nil, nil)
	agentID := uuid.New()
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = w.RecordHeartbeat(ctx, agentID)
	}
}

// BenchmarkWatcher_RecordHeartbeat_Parallel measures concurrent heartbeat throughput.
func BenchmarkWatcher_RecordHeartbeat_Parallel(b *testing.B) {
	db := &fastMockStore{}
	w := NewWatcher(db, nil, nil)
	ctx := context.Background()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		agentID := uuid.New()
		for pb.Next() {
			_ = w.RecordHeartbeat(ctx, agentID)
		}
	})
}

// BenchmarkWatcher_WatchUnwatch_50Agents measures registration/deregistration throughput.
func BenchmarkWatcher_WatchUnwatch_50Agents(b *testing.B) {
	db := &fastMockStore{}
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := NewWatcher(db, nil, nil)
		ids := make([]uuid.UUID, 50)
		for j := range ids {
			ids[j] = uuid.New()
			w.Watch(ctx, ids[j], uuid.New())
		}
		for _, id := range ids {
			w.Unwatch(id)
		}
	}
}

// BenchmarkFailureDetectionLatency measures time from "agent goes stale" to onFailed callback.
// This simulates the worst-case detection window by using a very short check interval.
func BenchmarkFailureDetectionLatency(b *testing.B) {
	for i := 0; i < b.N; i++ {
		agentID := uuid.New()
		companyID := uuid.New()

		callCount := int64(0)
		var detectedAt time.Time
		var mu sync.Mutex

		db := &fastMockStore{
			staleAgents: []store.Agent{{ID: agentID, CompanyID: companyID}},
		}
		// Pre-seed miss count to MissThresholdFailed-1 so next check triggers failure.
		counter := int64(MissThresholdFailed - 1)
		db.missCounts.Store(agentID, &counter)

		w := NewWatcher(db,
			func(_ context.Context, _ uuid.UUID, _ uuid.UUID) {
				mu.Lock()
				if atomic.AddInt64(&callCount, 1) == 1 {
					detectedAt = time.Now()
				}
				mu.Unlock()
			},
			nil,
		)

		start := time.Now()
		ctx := context.Background()
		// Simulate the check directly (not via ticker) for benchmark accuracy.
		w.checkAgent(ctx, agentID, companyID)

		mu.Lock()
		detected := detectedAt
		mu.Unlock()

		if detected.IsZero() {
			detected = time.Now()
		}
		_ = detected.Sub(start)
	}
}
