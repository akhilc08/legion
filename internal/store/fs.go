package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

func (db *DB) GrantFSPermission(ctx context.Context, agentID uuid.UUID, path string, level PermissionLevel, grantedBy *uuid.UUID) error {
	_, err := db.Pool.Exec(ctx,
		`INSERT INTO fs_permissions (agent_id, path, permission_level, granted_by)
		 VALUES ($1,$2,$3,$4)
		 ON CONFLICT (agent_id, path) DO UPDATE SET permission_level = $3, granted_by = $4`,
		agentID, path, level, grantedBy,
	)
	return err
}

func (db *DB) RevokeFSPermission(ctx context.Context, agentID uuid.UUID, path string) error {
	_, err := db.Pool.Exec(ctx,
		`DELETE FROM fs_permissions WHERE agent_id = $1 AND path = $2`, agentID, path,
	)
	return err
}

func (db *DB) ListFSPermissions(ctx context.Context, agentID uuid.UUID) ([]FSPermission, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT id, agent_id, path, permission_level, granted_by FROM fs_permissions WHERE agent_id = $1`,
		agentID,
	)
	if err != nil {
		return nil, fmt.Errorf("list fs perms: %w", err)
	}
	defer rows.Close()

	var perms []FSPermission
	for rows.Next() {
		var p FSPermission
		if err := rows.Scan(&p.ID, &p.AgentID, &p.Path, &p.PermissionLevel, &p.GrantedBy); err != nil {
			return nil, err
		}
		perms = append(perms, p)
	}
	return perms, nil
}

// CheckFSPermission returns the effective permission level for an agent on a path.
// It walks from the exact path up to parent directories, taking the most permissive match.
func (db *DB) CheckFSPermission(ctx context.Context, agentID uuid.UUID, path string) (PermissionLevel, bool, error) {
	perms, err := db.ListFSPermissions(ctx, agentID)
	if err != nil {
		return "", false, err
	}

	best := ""
	rank := map[PermissionLevel]int{PermRead: 1, PermWrite: 2, PermAdmin: 3}

	for _, p := range perms {
		if strings.HasPrefix(path, p.Path) || path == p.Path {
			if rank[p.PermissionLevel] > rank[PermissionLevel(best)] {
				best = string(p.PermissionLevel)
			}
		}
	}

	if best == "" {
		return "", false, nil
	}
	return PermissionLevel(best), true, nil
}

// CascadePermissionsFromManager copies manager's permissions to a newly restructured agent.
func (db *DB) CascadePermissionsFromManager(ctx context.Context, agentID, managerID uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO fs_permissions (agent_id, path, permission_level, granted_by)
		SELECT $1, path, permission_level, $2
		FROM fs_permissions
		WHERE agent_id = $2
		ON CONFLICT (agent_id, path) DO NOTHING
	`, agentID, managerID)
	return err
}

// RevokeFSPermissionByID deletes a permission row by its UUID primary key.
func (db *DB) RevokeFSPermissionByID(ctx context.Context, permID uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM fs_permissions WHERE id = $1`, permID)
	return err
}

// ListFSPermissionsForCompany returns all fs_permissions for all agents in a company (for FS browser).
func (db *DB) ListFSPermissionsForCompany(ctx context.Context, companyID uuid.UUID) ([]FSPermission, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT fp.id, fp.agent_id, fp.path, fp.permission_level, fp.granted_by
		 FROM fs_permissions fp
		 JOIN agents a ON a.id = fp.agent_id
		 WHERE a.company_id = $1`,
		companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("list company fs perms: %w", err)
	}
	defer rows.Close()

	var perms []FSPermission
	for rows.Next() {
		var p FSPermission
		if err := rows.Scan(&p.ID, &p.AgentID, &p.Path, &p.PermissionLevel, &p.GrantedBy); err != nil {
			return nil, err
		}
		perms = append(perms, p)
	}
	return perms, nil
}
