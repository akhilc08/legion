package store_test

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v2"
)

var userColumns = []string{"id", "email", "password_hash", "created_at"}

// --- CreateUser ---

func TestCreateUser_Success(t *testing.T) {
	mock, db := newMock(t)
	id := mustUUID(t)
	ts := fixedTime()

	mock.ExpectQuery(`INSERT INTO users`).
		WithArgs("alice@example.com", "hashed").
		WillReturnRows(mock.NewRows(userColumns).AddRow(id, "alice@example.com", "hashed", ts))

	u, err := db.CreateUser(context.Background(), "alice@example.com", "hashed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Email != "alice@example.com" {
		t.Errorf("expected email alice@example.com, got %q", u.Email)
	}
	if u.PasswordHash != "hashed" {
		t.Errorf("expected hash 'hashed', got %q", u.PasswordHash)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestCreateUser_DBError(t *testing.T) {
	mock, db := newMock(t)

	mock.ExpectQuery(`INSERT INTO users`).
		WillReturnError(errors.New("unique violation"))

	u, err := db.CreateUser(context.Background(), "dup@example.com", "hash")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if u != nil {
		t.Errorf("expected nil user on error")
	}
}

// --- GetUserByEmail ---

func TestGetUserByEmail_Success(t *testing.T) {
	mock, db := newMock(t)
	id := mustUUID(t)
	ts := fixedTime()

	mock.ExpectQuery(`SELECT .* FROM users WHERE email`).
		WithArgs("alice@example.com").
		WillReturnRows(mock.NewRows(userColumns).AddRow(id, "alice@example.com", "hashed", ts))

	u, err := db.GetUserByEmail(context.Background(), "alice@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.ID != id {
		t.Errorf("expected ID %v, got %v", id, u.ID)
	}
}

func TestGetUserByEmail_NotFound(t *testing.T) {
	mock, db := newMock(t)

	mock.ExpectQuery(`SELECT .* FROM users WHERE email`).
		WithArgs("missing@example.com").
		WillReturnError(errors.New("no rows in result set"))

	u, err := db.GetUserByEmail(context.Background(), "missing@example.com")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if u != nil {
		t.Errorf("expected nil user on error")
	}
}

// --- GetUser ---

func TestGetUser_Success(t *testing.T) {
	mock, db := newMock(t)
	id := mustUUID(t)
	ts := fixedTime()

	mock.ExpectQuery(`SELECT .* FROM users WHERE id`).
		WithArgs(id).
		WillReturnRows(mock.NewRows(userColumns).AddRow(id, "bob@example.com", "hash2", ts))

	u, err := db.GetUser(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Email != "bob@example.com" {
		t.Errorf("expected bob@example.com, got %q", u.Email)
	}
}

func TestGetUser_NotFound(t *testing.T) {
	mock, db := newMock(t)
	id := mustUUID(t)

	mock.ExpectQuery(`SELECT .* FROM users WHERE id`).
		WithArgs(id).
		WillReturnError(errors.New("no rows"))

	u, err := db.GetUser(context.Background(), id)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if u != nil {
		t.Error("expected nil user")
	}
}

// --- AddUserToCompany ---

func TestAddUserToCompany_Success(t *testing.T) {
	mock, db := newMock(t)
	userID := mustUUID(t)
	companyID := mustUUID(t)

	mock.ExpectExec(`INSERT INTO user_companies`).
		WithArgs(userID, companyID, "admin").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err := db.AddUserToCompany(context.Background(), userID, companyID, "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestAddUserToCompany_Error(t *testing.T) {
	mock, db := newMock(t)
	userID := mustUUID(t)
	companyID := mustUUID(t)

	mock.ExpectExec(`INSERT INTO user_companies`).
		WillReturnError(errors.New("foreign key violation"))

	err := db.AddUserToCompany(context.Background(), userID, companyID, "member")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ON CONFLICT DO NOTHING means a re-insert is silently ignored.
func TestAddUserToCompany_ConflictIgnored(t *testing.T) {
	mock, db := newMock(t)
	userID := mustUUID(t)
	companyID := mustUUID(t)

	// Simulates ON CONFLICT DO NOTHING: 0 rows affected but no error.
	mock.ExpectExec(`INSERT INTO user_companies`).
		WithArgs(userID, companyID, "member").
		WillReturnResult(pgxmock.NewResult("INSERT", 0))

	err := db.AddUserToCompany(context.Background(), userID, companyID, "member")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- UserCanAccessCompany ---

func TestUserCanAccessCompany_True(t *testing.T) {
	mock, db := newMock(t)
	userID := mustUUID(t)
	companyID := mustUUID(t)

	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(userID, companyID).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))

	ok, err := db.UserCanAccessCompany(context.Background(), userID, companyID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected true, got false")
	}
}

func TestUserCanAccessCompany_False(t *testing.T) {
	mock, db := newMock(t)
	userID := mustUUID(t)
	companyID := mustUUID(t)

	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(userID, companyID).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(false))

	ok, err := db.UserCanAccessCompany(context.Background(), userID, companyID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected false, got true")
	}
}

func TestUserCanAccessCompany_Error(t *testing.T) {
	mock, db := newMock(t)
	userID := mustUUID(t)
	companyID := mustUUID(t)

	mock.ExpectQuery(`SELECT EXISTS`).
		WillReturnError(errors.New("query error"))

	_, err := db.UserCanAccessCompany(context.Background(), userID, companyID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
