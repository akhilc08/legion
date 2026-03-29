package store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

func (db *DB) UpsertHeartbeat(ctx context.Context, agentID uuid.UUID) error {
	_, err := db.Pool.Exec(ctx,
		`INSERT INTO heartbeats (agent_id, last_seen_at, consecutive_misses)
		 VALUES ($1, NOW(), 0)
		 ON CONFLICT (agent_id) DO UPDATE
		 SET last_seen_at = NOW(), consecutive_misses = 0`,
		agentID,
	)
	return err
}

func (db *DB) GetHeartbeat(ctx context.Context, agentID uuid.UUID) (*Heartbeat, error) {
	h := &Heartbeat{}
	err := db.Pool.QueryRow(ctx,
		`SELECT agent_id, last_seen_at, consecutive_misses FROM heartbeats WHERE agent_id = $1`, agentID,
	).Scan(&h.AgentID, &h.LastSeenAt, &h.ConsecutiveMisses)
	if err != nil {
		return nil, fmt.Errorf("get heartbeat: %w", err)
	}
	return h, nil
}

func (db *DB) IncrementMisses(ctx context.Context, agentID uuid.UUID) (int, error) {
	var misses int
	err := db.Pool.QueryRow(ctx,
		`INSERT INTO heartbeats (agent_id, last_seen_at, consecutive_misses)
		 VALUES ($1, NOW(), 1)
		 ON CONFLICT (agent_id) DO UPDATE
		 SET consecutive_misses = heartbeats.consecutive_misses + 1
		 RETURNING consecutive_misses`,
		agentID,
	).Scan(&misses)
	return misses, err
}

// StaleAgents returns agents whose heartbeat last_seen_at is older than threshold.
func (db *DB) StaleAgents(ctx context.Context, threshold time.Duration) ([]Agent, error) {
	cutoff := time.Now().Add(-threshold)
	rows, err := db.Pool.Query(ctx,
		`SELECT a.id, a.company_id, a.role, a.title, a.system_prompt, a.manager_id, a.runtime, a.status,
		        a.monthly_budget, a.token_spend, a.chat_token_spend, a.pid, a.created_at, a.updated_at
		 FROM agents a
		 JOIN heartbeats h ON h.agent_id = a.id
		 WHERE a.status IN ('working','idle') AND h.last_seen_at < $1`,
		cutoff,
	)
	if err != nil {
		return nil, fmt.Errorf("stale agents: %w", err)
	}
	defer rows.Close()

	var agents []Agent
	for rows.Next() {
		var a Agent
		if err := rows.Scan(
			&a.ID, &a.CompanyID, &a.Role, &a.Title, &a.SystemPrompt,
			&a.ManagerID, &a.Runtime, &a.Status,
			&a.MonthlyBudget, &a.TokenSpend, &a.ChatTokenSpend, &a.PID,
			&a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, err
		}
		agents = append(agents, a)
	}
	return agents, nil
}
