package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (db *DB) CreateCompany(ctx context.Context, name, goal string) (*Company, error) {
	c := &Company{}
	err := db.Pool.QueryRow(ctx,
		`INSERT INTO companies (name, goal) VALUES ($1, $2)
		 RETURNING id, name, goal, created_at`,
		name, goal,
	).Scan(&c.ID, &c.Name, &c.Goal, &c.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create company: %w", err)
	}
	return c, nil
}

func (db *DB) GetCompany(ctx context.Context, id uuid.UUID) (*Company, error) {
	c := &Company{}
	err := db.Pool.QueryRow(ctx,
		`SELECT id, name, goal, created_at FROM companies WHERE id = $1`, id,
	).Scan(&c.ID, &c.Name, &c.Goal, &c.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get company: %w", err)
	}
	return c, nil
}

func (db *DB) ListCompanies(ctx context.Context) ([]Company, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT id, name, goal, created_at FROM companies ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list companies: %w", err)
	}
	defer rows.Close()

	var companies []Company
	for rows.Next() {
		var c Company
		if err := rows.Scan(&c.ID, &c.Name, &c.Goal, &c.CreatedAt); err != nil {
			return nil, err
		}
		companies = append(companies, c)
	}
	return companies, nil
}

func (db *DB) UpdateCompanyGoal(ctx context.Context, id uuid.UUID, goal string) error {
	_, err := db.Pool.Exec(ctx,
		`UPDATE companies SET goal = $1 WHERE id = $2`, goal, id,
	)
	return err
}

func (db *DB) DeleteCompany(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM companies WHERE id = $1`, id)
	return err
}

// ListCompaniesForUser returns companies accessible to a user.
func (db *DB) ListCompaniesForUser(ctx context.Context, userID uuid.UUID) ([]Company, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT c.id, c.name, c.goal, c.created_at
		 FROM companies c
		 JOIN user_companies uc ON uc.company_id = c.id
		 WHERE uc.user_id = $1
		 ORDER BY c.created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list user companies: %w", err)
	}
	defer rows.Close()

	var companies []Company
	for rows.Next() {
		var c Company
		if err := rows.Scan(&c.ID, &c.Name, &c.Goal, &c.CreatedAt); err != nil {
			return nil, err
		}
		companies = append(companies, c)
	}
	return companies, nil
}
