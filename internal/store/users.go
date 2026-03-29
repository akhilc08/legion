package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (db *DB) CreateUser(ctx context.Context, email, passwordHash string) (*User, error) {
	u := &User{}
	err := db.Pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash) VALUES ($1,$2)
		 RETURNING id, email, password_hash, created_at`,
		email, passwordHash,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}

func (db *DB) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	u := &User{}
	err := db.Pool.QueryRow(ctx,
		`SELECT id, email, password_hash, created_at FROM users WHERE email = $1`, email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return u, nil
}

func (db *DB) GetUser(ctx context.Context, id uuid.UUID) (*User, error) {
	u := &User{}
	err := db.Pool.QueryRow(ctx,
		`SELECT id, email, password_hash, created_at FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return u, nil
}

func (db *DB) AddUserToCompany(ctx context.Context, userID, companyID uuid.UUID, role string) error {
	_, err := db.Pool.Exec(ctx,
		`INSERT INTO user_companies (user_id, company_id, role) VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`,
		userID, companyID, role,
	)
	return err
}

// UserCanAccessCompany checks if a user has access to a company.
func (db *DB) UserCanAccessCompany(ctx context.Context, userID, companyID uuid.UUID) (bool, error) {
	var exists bool
	err := db.Pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM user_companies WHERE user_id = $1 AND company_id = $2)`,
		userID, companyID,
	).Scan(&exists)
	return exists, err
}
