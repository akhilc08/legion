package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (db *DB) CreateIssue(ctx context.Context, issue *Issue) (*Issue, error) {
	out := &Issue{}
	err := db.Pool.QueryRow(ctx,
		`INSERT INTO issues (company_id, title, description, assignee_id, parent_id, created_by)
		 VALUES ($1,$2,$3,$4,$5,$6)
		 RETURNING id, company_id, title, description, assignee_id, parent_id, status,
		           output_path, created_by, attempt_count, last_failure_reason, escalation_id,
		           created_at, updated_at`,
		issue.CompanyID, issue.Title, issue.Description, issue.AssigneeID, issue.ParentID, issue.CreatedBy,
	).Scan(
		&out.ID, &out.CompanyID, &out.Title, &out.Description, &out.AssigneeID, &out.ParentID,
		&out.Status, &out.OutputPath, &out.CreatedBy, &out.AttemptCount,
		&out.LastFailureReason, &out.EscalationID, &out.CreatedAt, &out.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create issue: %w", err)
	}
	return out, nil
}

func (db *DB) GetIssue(ctx context.Context, id uuid.UUID) (*Issue, error) {
	out := &Issue{}
	err := db.Pool.QueryRow(ctx,
		`SELECT id, company_id, title, description, assignee_id, parent_id, status,
		        output_path, created_by, attempt_count, last_failure_reason, escalation_id,
		        created_at, updated_at
		 FROM issues WHERE id = $1`, id,
	).Scan(
		&out.ID, &out.CompanyID, &out.Title, &out.Description, &out.AssigneeID, &out.ParentID,
		&out.Status, &out.OutputPath, &out.CreatedBy, &out.AttemptCount,
		&out.LastFailureReason, &out.EscalationID, &out.CreatedAt, &out.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get issue: %w", err)
	}
	return out, nil
}

func (db *DB) ListIssuesByCompany(ctx context.Context, companyID uuid.UUID) ([]Issue, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT id, company_id, title, description, assignee_id, parent_id, status,
		        output_path, created_by, attempt_count, last_failure_reason, escalation_id,
		        created_at, updated_at
		 FROM issues WHERE company_id = $1 ORDER BY created_at ASC`, companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("list issues: %w", err)
	}
	defer rows.Close()
	return scanIssues(rows)
}

func (db *DB) ListReadyIssues(ctx context.Context, companyID uuid.UUID) ([]Issue, error) {
	// Issues that are pending with no unresolved dependencies.
	rows, err := db.Pool.Query(ctx, `
		SELECT i.id, i.company_id, i.title, i.description, i.assignee_id, i.parent_id, i.status,
		       i.output_path, i.created_by, i.attempt_count, i.last_failure_reason, i.escalation_id,
		       i.created_at, i.updated_at
		FROM issues i
		WHERE i.company_id = $1
		  AND i.status = 'pending'
		  AND NOT EXISTS (
		    SELECT 1 FROM issue_dependencies d
		    JOIN issues dep ON dep.id = d.depends_on_issue_id
		    WHERE d.issue_id = i.id AND dep.status != 'done'
		  )
		ORDER BY i.created_at ASC
	`, companyID)
	if err != nil {
		return nil, fmt.Errorf("list ready issues: %w", err)
	}
	defer rows.Close()
	return scanIssues(rows)
}

func (db *DB) UpdateIssueStatus(ctx context.Context, id uuid.UUID, status IssueStatus) error {
	_, err := db.Pool.Exec(ctx,
		`UPDATE issues SET status = $1, updated_at = NOW() WHERE id = $2`, status, id,
	)
	return err
}

func (db *DB) UpdateIssueAssignee(ctx context.Context, id, assigneeID uuid.UUID) error {
	_, err := db.Pool.Exec(ctx,
		`UPDATE issues SET assignee_id = $1, updated_at = NOW() WHERE id = $2`, assigneeID, id,
	)
	return err
}

func (db *DB) UpdateIssueOutput(ctx context.Context, id uuid.UUID, outputPath string) error {
	_, err := db.Pool.Exec(ctx,
		`UPDATE issues SET output_path = $1, status = 'done', updated_at = NOW() WHERE id = $2`,
		outputPath, id,
	)
	return err
}

func (db *DB) IncrementAttemptCount(ctx context.Context, id uuid.UUID, reason string) error {
	_, err := db.Pool.Exec(ctx,
		`UPDATE issues SET attempt_count = attempt_count + 1, last_failure_reason = $1, updated_at = NOW() WHERE id = $2`,
		reason, id,
	)
	return err
}

// CheckoutIssue atomically acquires a Postgres advisory lock on the issue and marks it in_progress.
// Returns false if already locked by another session.
func (db *DB) CheckoutIssue(ctx context.Context, issueID uuid.UUID, agentID uuid.UUID) (bool, error) {
	// Use the lower 32 bits of the UUID as lock key (sufficient for uniqueness within a session).
	lockKey := int64(issueID[0])<<24 | int64(issueID[1])<<16 | int64(issueID[2])<<8 | int64(issueID[3])

	var locked bool
	err := db.Pool.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, lockKey).Scan(&locked)
	if err != nil {
		return false, fmt.Errorf("advisory lock: %w", err)
	}
	if !locked {
		return false, nil
	}

	tag, err := db.Pool.Exec(ctx,
		`UPDATE issues SET status = 'in_progress', assignee_id = $1, updated_at = NOW()
		 WHERE id = $2 AND status = 'pending'`, agentID, issueID,
	)
	if err != nil {
		return false, fmt.Errorf("checkout issue update: %w", err)
	}
	if tag.RowsAffected() == 0 {
		// Already taken
		db.Pool.Exec(ctx, `SELECT pg_advisory_unlock($1)`, lockKey) //nolint
		return false, nil
	}
	return true, nil
}

// AddDependency adds a dependency edge; returns error if it creates a cycle.
func (db *DB) AddDependency(ctx context.Context, issueID, dependsOnID uuid.UUID) error {
	// Cycle check: if dependsOnID is already reachable from issueID, reject.
	var exists bool
	err := db.Pool.QueryRow(ctx, `
		WITH RECURSIVE deps AS (
			SELECT depends_on_issue_id AS id FROM issue_dependencies WHERE issue_id = $1
			UNION ALL
			SELECT d.depends_on_issue_id FROM issue_dependencies d JOIN deps ON d.issue_id = deps.id
		)
		SELECT EXISTS(SELECT 1 FROM deps WHERE id = $2)
	`, dependsOnID, issueID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("cycle check: %w", err)
	}
	if exists {
		return fmt.Errorf("dependency would create a cycle")
	}

	_, err = db.Pool.Exec(ctx,
		`INSERT INTO issue_dependencies (issue_id, depends_on_issue_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`,
		issueID, dependsOnID,
	)
	return err
}

func (db *DB) RemoveDependency(ctx context.Context, issueID, dependsOnID uuid.UUID) error {
	_, err := db.Pool.Exec(ctx,
		`DELETE FROM issue_dependencies WHERE issue_id = $1 AND depends_on_issue_id = $2`,
		issueID, dependsOnID,
	)
	return err
}

func (db *DB) GetDependencies(ctx context.Context, issueID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT depends_on_issue_id FROM issue_dependencies WHERE issue_id = $1`, issueID,
	)
	if err != nil {
		return nil, err
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

func scanIssues(rows pgx.Rows) ([]Issue, error) {
	var issues []Issue
	for rows.Next() {
		var i Issue
		if err := rows.Scan(
			&i.ID, &i.CompanyID, &i.Title, &i.Description, &i.AssigneeID, &i.ParentID,
			&i.Status, &i.OutputPath, &i.CreatedBy, &i.AttemptCount,
			&i.LastFailureReason, &i.EscalationID, &i.CreatedAt, &i.UpdatedAt,
		); err != nil {
			return nil, err
		}
		issues = append(issues, i)
	}
	return issues, nil
}
