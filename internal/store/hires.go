package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (db *DB) CreatePendingHire(ctx context.Context, h *PendingHire) (*PendingHire, error) {
	out := &PendingHire{}
	err := db.Pool.QueryRow(ctx,
		`INSERT INTO pending_hires
		 (company_id, requested_by_agent_id, role_title, reporting_to_agent_id,
		  system_prompt, runtime, budget_allocation, initial_task)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		 RETURNING id, company_id, requested_by_agent_id, role_title, reporting_to_agent_id,
		           system_prompt, runtime, budget_allocation, initial_task, status, created_at`,
		h.CompanyID, h.RequestedByAgentID, h.RoleTitle, h.ReportingToAgentID,
		h.SystemPrompt, h.Runtime, h.BudgetAllocation, h.InitialTask,
	).Scan(
		&out.ID, &out.CompanyID, &out.RequestedByAgentID, &out.RoleTitle, &out.ReportingToAgentID,
		&out.SystemPrompt, &out.Runtime, &out.BudgetAllocation, &out.InitialTask,
		&out.Status, &out.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create pending hire: %w", err)
	}
	return out, nil
}

func (db *DB) GetPendingHire(ctx context.Context, id uuid.UUID) (*PendingHire, error) {
	out := &PendingHire{}
	err := db.Pool.QueryRow(ctx,
		`SELECT id, company_id, requested_by_agent_id, role_title, reporting_to_agent_id,
		        system_prompt, runtime, budget_allocation, initial_task, status, created_at
		 FROM pending_hires WHERE id = $1`, id,
	).Scan(
		&out.ID, &out.CompanyID, &out.RequestedByAgentID, &out.RoleTitle, &out.ReportingToAgentID,
		&out.SystemPrompt, &out.Runtime, &out.BudgetAllocation, &out.InitialTask,
		&out.Status, &out.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get pending hire: %w", err)
	}
	return out, nil
}

func (db *DB) ListPendingHires(ctx context.Context, companyID uuid.UUID) ([]PendingHire, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT id, company_id, requested_by_agent_id, role_title, reporting_to_agent_id,
		        system_prompt, runtime, budget_allocation, initial_task, status, created_at
		 FROM pending_hires WHERE company_id = $1 AND status = 'pending' ORDER BY created_at DESC`,
		companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("list pending hires: %w", err)
	}
	defer rows.Close()

	var hires []PendingHire
	for rows.Next() {
		var h PendingHire
		if err := rows.Scan(
			&h.ID, &h.CompanyID, &h.RequestedByAgentID, &h.RoleTitle, &h.ReportingToAgentID,
			&h.SystemPrompt, &h.Runtime, &h.BudgetAllocation, &h.InitialTask,
			&h.Status, &h.CreatedAt,
		); err != nil {
			return nil, err
		}
		hires = append(hires, h)
	}
	return hires, nil
}

func (db *DB) UpdateHireStatus(ctx context.Context, id uuid.UUID, status HireStatus) error {
	_, err := db.Pool.Exec(ctx,
		`UPDATE pending_hires SET status = $1 WHERE id = $2`, status, id,
	)
	return err
}
