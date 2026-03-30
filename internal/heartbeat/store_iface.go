package heartbeat

import (
	"context"
	"time"

	"conductor/internal/store"
	"github.com/google/uuid"
)

// StoreIface is the subset of *store.DB methods used by the Watcher.
// Using an interface makes the watcher testable without a real database.
type StoreIface interface {
	UpsertHeartbeat(ctx context.Context, agentID uuid.UUID) error
	StaleAgents(ctx context.Context, threshold time.Duration) ([]store.Agent, error)
	IncrementMisses(ctx context.Context, agentID uuid.UUID) (int, error)
	UpdateAgentStatus(ctx context.Context, id uuid.UUID, status store.AgentStatus) error
}
