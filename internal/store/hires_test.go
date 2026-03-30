package store_test

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v2"

	"conductor/internal/store"
)

var hireColumns = []string{
	"id", "company_id", "requested_by_agent_id", "role_title", "reporting_to_agent_id",
	"system_prompt", "runtime", "budget_allocation", "initial_task", "status", "created_at",
}

func sampleHire(t *testing.T) *store.PendingHire {
	t.Helper()
	task := "Build the auth module"
	return &store.PendingHire{
		ID:                 mustUUID(t),
		CompanyID:          mustUUID(t),
		RequestedByAgentID: mustUUID(t),
		RoleTitle:          "Backend Engineer",
		ReportingToAgentID: mustUUID(t),
		SystemPrompt:       "You are a backend engineer.",
		Runtime:            store.RuntimeClaudeCode,
		BudgetAllocation:   500,
		InitialTask:        &task,
		Status:             store.HirePending,
		CreatedAt:          fixedTime(),
	}
}

func hireRow(mock pgxmock.PgxPoolIface, h *store.PendingHire) *pgxmock.Rows {
	return mock.NewRows(hireColumns).AddRow(
		h.ID, h.CompanyID, h.RequestedByAgentID, h.RoleTitle, h.ReportingToAgentID,
		h.SystemPrompt, h.Runtime, h.BudgetAllocation, h.InitialTask,
		h.Status, h.CreatedAt,
	)
}

// --- CreatePendingHire ---

func TestCreatePendingHire_Success(t *testing.T) {
	mock, db := newMock(t)
	h := sampleHire(t)

	mock.ExpectQuery(`INSERT INTO pending_hires`).
		WithArgs(
			h.CompanyID, h.RequestedByAgentID, h.RoleTitle, h.ReportingToAgentID,
			h.SystemPrompt, h.Runtime, h.BudgetAllocation, h.InitialTask,
		).
		WillReturnRows(hireRow(mock, h))

	got, err := db.CreatePendingHire(context.Background(), h)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != h.ID {
		t.Errorf("expected ID %v, got %v", h.ID, got.ID)
	}
	if got.RoleTitle != "Backend Engineer" {
		t.Errorf("expected role title 'Backend Engineer', got %q", got.RoleTitle)
	}
	if got.InitialTask == nil || *got.InitialTask != "Build the auth module" {
		t.Errorf("expected initial task 'Build the auth module', got %v", got.InitialTask)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestCreatePendingHire_NoInitialTask(t *testing.T) {
	mock, db := newMock(t)
	h := sampleHire(t)
	h.InitialTask = nil

	mock.ExpectQuery(`INSERT INTO pending_hires`).
		WithArgs(
			h.CompanyID, h.RequestedByAgentID, h.RoleTitle, h.ReportingToAgentID,
			h.SystemPrompt, h.Runtime, h.BudgetAllocation, (*string)(nil),
		).
		WillReturnRows(hireRow(mock, h))

	got, err := db.CreatePendingHire(context.Background(), h)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.InitialTask != nil {
		t.Errorf("expected nil initial task, got %v", got.InitialTask)
	}
}

func TestCreatePendingHire_Error(t *testing.T) {
	mock, db := newMock(t)
	h := sampleHire(t)

	mock.ExpectQuery(`INSERT INTO pending_hires`).
		WillReturnError(errors.New("db error"))

	got, err := db.CreatePendingHire(context.Background(), h)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got != nil {
		t.Error("expected nil on error")
	}
}

// --- GetPendingHire ---

func TestGetPendingHire_Success(t *testing.T) {
	mock, db := newMock(t)
	h := sampleHire(t)

	mock.ExpectQuery(`SELECT .* FROM pending_hires WHERE id`).
		WithArgs(h.ID).
		WillReturnRows(hireRow(mock, h))

	got, err := db.GetPendingHire(context.Background(), h.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != h.ID {
		t.Errorf("expected ID %v, got %v", h.ID, got.ID)
	}
}

func TestGetPendingHire_NotFound(t *testing.T) {
	mock, db := newMock(t)
	id := mustUUID(t)

	mock.ExpectQuery(`SELECT .* FROM pending_hires WHERE id`).
		WithArgs(id).
		WillReturnError(errors.New("no rows"))

	got, err := db.GetPendingHire(context.Background(), id)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got != nil {
		t.Error("expected nil hire")
	}
}

// --- ListPendingHires ---

func TestListPendingHires_Success(t *testing.T) {
	mock, db := newMock(t)
	h := sampleHire(t)

	mock.ExpectQuery(`SELECT .* FROM pending_hires WHERE company_id .* AND status = 'pending'`).
		WithArgs(h.CompanyID).
		WillReturnRows(hireRow(mock, h))

	hires, err := db.ListPendingHires(context.Background(), h.CompanyID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hires) != 1 {
		t.Errorf("expected 1 hire, got %d", len(hires))
	}
	if hires[0].RoleTitle != h.RoleTitle {
		t.Errorf("expected %q, got %q", h.RoleTitle, hires[0].RoleTitle)
	}
}

func TestListPendingHires_Empty(t *testing.T) {
	mock, db := newMock(t)
	companyID := mustUUID(t)

	mock.ExpectQuery(`SELECT .* FROM pending_hires WHERE company_id .* AND status = 'pending'`).
		WithArgs(companyID).
		WillReturnRows(mock.NewRows(hireColumns))

	hires, err := db.ListPendingHires(context.Background(), companyID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hires) != 0 {
		t.Errorf("expected 0 hires, got %d", len(hires))
	}
}

func TestListPendingHires_Error(t *testing.T) {
	mock, db := newMock(t)
	companyID := mustUUID(t)

	mock.ExpectQuery(`SELECT .* FROM pending_hires WHERE company_id .* AND status = 'pending'`).
		WillReturnError(errors.New("query error"))

	_, err := db.ListPendingHires(context.Background(), companyID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- UpdateHireStatus ---

func TestUpdateHireStatus_Success(t *testing.T) {
	mock, db := newMock(t)
	id := mustUUID(t)

	mock.ExpectExec(`UPDATE pending_hires SET status`).
		WithArgs(store.HireApproved, id).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := db.UpdateHireStatus(context.Background(), id, store.HireApproved)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestUpdateHireStatus_Rejected(t *testing.T) {
	mock, db := newMock(t)
	id := mustUUID(t)

	mock.ExpectExec(`UPDATE pending_hires SET status`).
		WithArgs(store.HireRejected, id).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := db.UpdateHireStatus(context.Background(), id, store.HireRejected)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateHireStatus_Error(t *testing.T) {
	mock, db := newMock(t)
	id := mustUUID(t)

	mock.ExpectExec(`UPDATE pending_hires SET status`).
		WillReturnError(errors.New("update error"))

	err := db.UpdateHireStatus(context.Background(), id, store.HireApproved)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// Verify HireStatus constants.
func TestHireStatusConstants(t *testing.T) {
	cases := []struct {
		val  store.HireStatus
		want string
	}{
		{store.HirePending, "pending"},
		{store.HireApproved, "approved"},
		{store.HireRejected, "rejected"},
	}
	for _, c := range cases {
		if string(c.val) != c.want {
			t.Errorf("expected %q, got %q", c.want, c.val)
		}
	}
}
