// Package heartbeat monitors agent liveness and triggers failure handling
// when agents miss consecutive heartbeats.
package heartbeat

import (
	"context"
	"log"
	"sync"
	"time"

	"conductor/internal/store"
	"github.com/google/uuid"
)

const (
	// HeartbeatInterval is how often each agent should send a ping.
	HeartbeatInterval = 15 * time.Second

	// CheckInterval is how often the watcher polls for stale agents.
	CheckInterval = 20 * time.Second

	// MissThresholdDegraded is the number of misses before marking DEGRADED.
	MissThresholdDegraded = 1

	// MissThresholdFailed is the number of consecutive misses before eviction.
	MissThresholdFailed = 3

	// StaleThreshold is the time window used to detect missed heartbeats.
	StaleThreshold = HeartbeatInterval + 5*time.Second
)

// FailureHandler is called when an agent reaches the failure threshold.
// The orchestrator uses this to kill, respawn, and release the agent's issue.
type FailureHandler func(ctx context.Context, agentID uuid.UUID, companyID uuid.UUID)

// DegradedHandler is called when an agent misses its first heartbeat.
type DegradedHandler func(ctx context.Context, agentID uuid.UUID, companyID uuid.UUID)

// Watcher runs a goroutine-per-agent heartbeat monitor.
type Watcher struct {
	db             *store.DB
	onFailed       FailureHandler
	onDegraded     DegradedHandler
	mu             sync.Mutex
	watching       map[uuid.UUID]context.CancelFunc
}

func NewWatcher(db *store.DB, onFailed FailureHandler, onDegraded DegradedHandler) *Watcher {
	return &Watcher{
		db:         db,
		onFailed:   onFailed,
		onDegraded: onDegraded,
		watching:   make(map[uuid.UUID]context.CancelFunc),
	}
}

// Watch starts monitoring an agent. Safe to call multiple times for the same agent.
func (w *Watcher) Watch(ctx context.Context, agentID uuid.UUID, companyID uuid.UUID) {
	w.mu.Lock()
	if _, exists := w.watching[agentID]; exists {
		w.mu.Unlock()
		return
	}
	watchCtx, cancel := context.WithCancel(ctx)
	w.watching[agentID] = cancel
	w.mu.Unlock()

	go w.watchAgent(watchCtx, agentID, companyID)
}

// Unwatch stops monitoring an agent.
func (w *Watcher) Unwatch(agentID uuid.UUID) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if cancel, ok := w.watching[agentID]; ok {
		cancel()
		delete(w.watching, agentID)
	}
}

// RecordHeartbeat updates the agent's last-seen timestamp.
func (w *Watcher) RecordHeartbeat(ctx context.Context, agentID uuid.UUID) error {
	return w.db.UpsertHeartbeat(ctx, agentID)
}

func (w *Watcher) watchAgent(ctx context.Context, agentID uuid.UUID, companyID uuid.UUID) {
	ticker := time.NewTicker(CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.checkAgent(ctx, agentID, companyID)
		}
	}
}

func (w *Watcher) checkAgent(ctx context.Context, agentID, companyID uuid.UUID) {
	stale, err := w.db.StaleAgents(ctx, StaleThreshold)
	if err != nil {
		log.Printf("heartbeat: stale agent query: %v", err)
		return
	}

	for _, a := range stale {
		if a.ID != agentID {
			continue
		}

		misses, err := w.db.IncrementMisses(ctx, agentID)
		if err != nil {
			log.Printf("heartbeat: increment misses for %s: %v", agentID, err)
			continue
		}

		log.Printf("heartbeat: agent %s missed %d/%d", agentID, misses, MissThresholdFailed)

		if misses >= MissThresholdFailed {
			log.Printf("heartbeat: agent %s FAILED — evicting", agentID)
			if err := w.db.UpdateAgentStatus(ctx, agentID, store.StatusFailed); err != nil {
				log.Printf("heartbeat: update status failed: %v", err)
			}
			w.Unwatch(agentID)
			if w.onFailed != nil {
				go w.onFailed(ctx, agentID, companyID)
			}
		} else if misses >= MissThresholdDegraded {
			if err := w.db.UpdateAgentStatus(ctx, agentID, store.StatusDegraded); err != nil {
				log.Printf("heartbeat: update status degraded: %v", err)
			}
			if w.onDegraded != nil {
				go w.onDegraded(ctx, agentID, companyID)
			}
		}
		return
	}
}
