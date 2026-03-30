package store_test

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v2"
)

var companyColumns = []string{"id", "name", "goal", "created_at"}

// --- CreateCompany ---

func TestCreateCompany_Success(t *testing.T) {
	mock, db := newMock(t)
	id := mustUUID(t)
	ts := fixedTime()

	mock.ExpectQuery(`INSERT INTO companies`).
		WithArgs("Acme Corp", "world domination").
		WillReturnRows(mock.NewRows(companyColumns).AddRow(id, "Acme Corp", "world domination", ts))

	c, err := db.CreateCompany(context.Background(), "Acme Corp", "world domination")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Name != "Acme Corp" {
		t.Errorf("expected 'Acme Corp', got %q", c.Name)
	}
	if c.Goal != "world domination" {
		t.Errorf("expected 'world domination', got %q", c.Goal)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestCreateCompany_DBError(t *testing.T) {
	mock, db := newMock(t)

	mock.ExpectQuery(`INSERT INTO companies`).
		WillReturnError(errors.New("db error"))

	c, err := db.CreateCompany(context.Background(), "Fail Corp", "fail")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if c != nil {
		t.Error("expected nil company on error")
	}
}

// --- GetCompany ---

func TestGetCompany_Success(t *testing.T) {
	mock, db := newMock(t)
	id := mustUUID(t)
	ts := fixedTime()

	mock.ExpectQuery(`SELECT .* FROM companies WHERE id`).
		WithArgs(id).
		WillReturnRows(mock.NewRows(companyColumns).AddRow(id, "Acme Corp", "goal", ts))

	c, err := db.GetCompany(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.ID != id {
		t.Errorf("expected ID %v, got %v", id, c.ID)
	}
}

func TestGetCompany_NotFound(t *testing.T) {
	mock, db := newMock(t)
	id := mustUUID(t)

	mock.ExpectQuery(`SELECT .* FROM companies WHERE id`).
		WithArgs(id).
		WillReturnError(errors.New("no rows"))

	c, err := db.GetCompany(context.Background(), id)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if c != nil {
		t.Error("expected nil company")
	}
}

// --- ListCompanies ---

func TestListCompanies_Success(t *testing.T) {
	mock, db := newMock(t)
	id1 := mustUUID(t)
	id2 := mustUUID(t)
	ts := fixedTime()

	mock.ExpectQuery(`SELECT .* FROM companies ORDER BY created_at DESC`).
		WillReturnRows(mock.NewRows(companyColumns).
			AddRow(id1, "Alpha", "goal1", ts).
			AddRow(id2, "Beta", "goal2", ts))

	companies, err := db.ListCompanies(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(companies) != 2 {
		t.Errorf("expected 2 companies, got %d", len(companies))
	}
}

func TestListCompanies_Empty(t *testing.T) {
	mock, db := newMock(t)

	mock.ExpectQuery(`SELECT .* FROM companies ORDER BY created_at DESC`).
		WillReturnRows(mock.NewRows(companyColumns))

	companies, err := db.ListCompanies(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(companies) != 0 {
		t.Errorf("expected 0 companies, got %d", len(companies))
	}
}

func TestListCompanies_QueryError(t *testing.T) {
	mock, db := newMock(t)

	mock.ExpectQuery(`SELECT .* FROM companies ORDER BY created_at DESC`).
		WillReturnError(errors.New("query error"))

	_, err := db.ListCompanies(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- UpdateCompanyGoal ---

func TestUpdateCompanyGoal_Success(t *testing.T) {
	mock, db := newMock(t)
	id := mustUUID(t)

	mock.ExpectExec(`UPDATE companies SET goal`).
		WithArgs("new goal", id).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := db.UpdateCompanyGoal(context.Background(), id, "new goal")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateCompanyGoal_Error(t *testing.T) {
	mock, db := newMock(t)
	id := mustUUID(t)

	mock.ExpectExec(`UPDATE companies SET goal`).
		WillReturnError(errors.New("update error"))

	err := db.UpdateCompanyGoal(context.Background(), id, "new goal")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- DeleteCompany ---

func TestDeleteCompany_Success(t *testing.T) {
	mock, db := newMock(t)
	id := mustUUID(t)

	mock.ExpectExec(`DELETE FROM companies WHERE id`).
		WithArgs(id).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err := db.DeleteCompany(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteCompany_Error(t *testing.T) {
	mock, db := newMock(t)
	id := mustUUID(t)

	mock.ExpectExec(`DELETE FROM companies WHERE id`).
		WillReturnError(errors.New("delete error"))

	err := db.DeleteCompany(context.Background(), id)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- ListCompaniesForUser ---

func TestListCompaniesForUser_Success(t *testing.T) {
	mock, db := newMock(t)
	userID := mustUUID(t)
	id := mustUUID(t)
	ts := fixedTime()

	mock.ExpectQuery(`SELECT c\.id, c\.name, c\.goal, c\.created_at`).
		WithArgs(userID).
		WillReturnRows(mock.NewRows(companyColumns).AddRow(id, "MyCompany", "my goal", ts))

	companies, err := db.ListCompaniesForUser(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(companies) != 1 {
		t.Errorf("expected 1 company, got %d", len(companies))
	}
	if companies[0].Name != "MyCompany" {
		t.Errorf("expected 'MyCompany', got %q", companies[0].Name)
	}
}

func TestListCompaniesForUser_Empty(t *testing.T) {
	mock, db := newMock(t)
	userID := mustUUID(t)

	mock.ExpectQuery(`SELECT c\.id, c\.name, c\.goal, c\.created_at`).
		WithArgs(userID).
		WillReturnRows(mock.NewRows(companyColumns))

	companies, err := db.ListCompaniesForUser(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(companies) != 0 {
		t.Errorf("expected 0 companies, got %d", len(companies))
	}
}

func TestListCompaniesForUser_Error(t *testing.T) {
	mock, db := newMock(t)
	userID := mustUUID(t)

	mock.ExpectQuery(`SELECT c\.id, c\.name, c\.goal, c\.created_at`).
		WillReturnError(errors.New("query error"))

	_, err := db.ListCompaniesForUser(context.Background(), userID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
