package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (db *DB) CreateAgent(ctx context.Context, a *Agent) (*Agent, error) {
	out := &Agent{}
	err := db.Pool.QueryRow(ctx,
		`INSERT INTO agents (company_id, role, title, system_prompt, manager_id, runtime, monthly_budget)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)
		 RETURNING id, company_id, role, title, system_prompt, manager_id, runtime, status,
		           monthly_budget, token_spend, chat_token_spend, pid, created_at, updated_at`,
		a.CompanyID, a.Role, a.Title, a.SystemPrompt, a.ManagerID, a.Runtime, a.MonthlyBudget,
	).Scan(
		&out.ID, &out.CompanyID, &out.Role, &out.Title, &out.SystemPrompt,
		&out.ManagerID, &out.Runtime, &out.Status,
		&out.MonthlyBudget, &out.TokenSpend, &out.ChatTokenSpend, &out.PID,
		&out.CreatedAt, &out.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create agent: %w", err)
	}
	return out, nil
}

func (db *DB) GetAgent(ctx context.Context, id uuid.UUID) (*Agent, error) {
	a := &Agent{}
	err := db.Pool.QueryRow(ctx,
		`SELECT id, company_id, role, title, system_prompt, manager_id, runtime, status,
		        monthly_budget, token_spend, chat_token_spend, pid, created_at, updated_at
		 FROM agents WHERE id = $1`, id,
	).Scan(
		&a.ID, &a.CompanyID, &a.Role, &a.Title, &a.SystemPrompt,
		&a.ManagerID, &a.Runtime, &a.Status,
		&a.MonthlyBudget, &a.TokenSpend, &a.ChatTokenSpend, &a.PID,
		&a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get agent: %w", err)
	}
	return a, nil
}

func (db *DB) ListAgentsByCompany(ctx context.Context, companyID uuid.UUID) ([]Agent, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT id, company_id, role, title, system_prompt, manager_id, runtime, status,
		        monthly_budget, token_spend, chat_token_spend, pid, created_at, updated_at
		 FROM agents WHERE company_id = $1 ORDER BY created_at ASC`, companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
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

func (db *DB) UpdateAgentStatus(ctx context.Context, id uuid.UUID, status AgentStatus) error {
	_, err := db.Pool.Exec(ctx,
		`UPDATE agents SET status = $1, updated_at = NOW() WHERE id = $2`, status, id,
	)
	return err
}

func (db *DB) UpdateAgentPID(ctx context.Context, id uuid.UUID, pid *int) error {
	_, err := db.Pool.Exec(ctx,
		`UPDATE agents SET pid = $1, updated_at = NOW() WHERE id = $2`, pid, id,
	)
	return err
}

func (db *DB) UpdateAgentManager(ctx context.Context, agentID, managerID uuid.UUID) error {
	_, err := db.Pool.Exec(ctx,
		`UPDATE agents SET manager_id = $1, updated_at = NOW() WHERE id = $2`, managerID, agentID,
	)
	return err
}

func (db *DB) AddTokenSpend(ctx context.Context, agentID uuid.UUID, tokens int, isChat bool) error {
	if isChat {
		_, err := db.Pool.Exec(ctx,
			`UPDATE agents SET chat_token_spend = chat_token_spend + $1, updated_at = NOW() WHERE id = $2`,
			tokens, agentID,
		)
		return err
	}
	_, err := db.Pool.Exec(ctx,
		`UPDATE agents SET token_spend = token_spend + $1, updated_at = NOW() WHERE id = $2`,
		tokens, agentID,
	)
	return err
}

func (db *DB) DeleteAgent(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM agents WHERE id = $1`, id)
	return err
}

// GetSubtreeAgentIDs returns all agent IDs in the subtree rooted at agentID (inclusive).
func (db *DB) GetSubtreeAgentIDs(ctx context.Context, agentID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := db.Pool.Query(ctx, `
		WITH RECURSIVE subtree AS (
			SELECT id FROM agents WHERE id = $1
			UNION ALL
			SELECT a.id FROM agents a
			JOIN subtree s ON a.manager_id = s.id
		)
		SELECT id FROM subtree
	`, agentID)
	if err != nil {
		return nil, fmt.Errorf("get subtree: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// GetCEO returns the root agent (no manager) for a company.
func (db *DB) GetCEO(ctx context.Context, companyID uuid.UUID) (*Agent, error) {
	a := &Agent{}
	err := db.Pool.QueryRow(ctx,
		`SELECT id, company_id, role, title, system_prompt, manager_id, runtime, status,
		        monthly_budget, token_spend, chat_token_spend, pid, created_at, updated_at
		 FROM agents WHERE company_id = $1 AND manager_id IS NULL LIMIT 1`, companyID,
	).Scan(
		&a.ID, &a.CompanyID, &a.Role, &a.Title, &a.SystemPrompt,
		&a.ManagerID, &a.Runtime, &a.Status,
		&a.MonthlyBudget, &a.TokenSpend, &a.ChatTokenSpend, &a.PID,
		&a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get CEO: %w", err)
	}
	return a, nil
}
