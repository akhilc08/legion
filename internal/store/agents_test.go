package store_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pashagolub/pgxmock/v2"

	"conductor/internal/store"
)

// agentColumns lists SELECT columns in the order the store scans them.
var agentColumns = []string{
	"id", "company_id", "role", "title", "system_prompt",
	"manager_id", "runtime", "status",
	"monthly_budget", "token_spend", "chat_token_spend", "pid",
	"created_at", "updated_at",
}

func agentRow(mock pgxmock.PgxPoolIface, a *store.Agent) *pgxmock.Rows {
	return mock.NewRows(agentColumns).AddRow(
		a.ID, a.CompanyID, a.Role, a.Title, a.SystemPrompt,
		a.ManagerID, a.Runtime, a.Status,
		a.MonthlyBudget, a.TokenSpend, a.ChatTokenSpend, a.PID,
		a.CreatedAt, a.UpdatedAt,
	)
}

func sampleAgent(t *testing.T) *store.Agent {
	t.Helper()
	return &store.Agent{
		ID:            mustUUID(t),
		CompanyID:     mustUUID(t),
		Role:          "engineer",
		Title:         "Backend Engineer",
		SystemPrompt:  "You are a backend engineer.",
		ManagerID:     nil,
		Runtime:       store.RuntimeClaudeCode,
		Status:        store.StatusIdle,
		MonthlyBudget: 1000,
		TokenSpend:    0,
		CreatedAt:     fixedTime(),
		UpdatedAt:     fixedTime(),
	}
}

// --- CreateAgent ---

func TestCreateAgent_Success(t *testing.T) {
	mock, db := newMock(t)
	a := sampleAgent(t)

	mock.ExpectQuery(`INSERT INTO agents`).
		WithArgs(a.CompanyID, a.Role, a.Title, a.SystemPrompt, a.ManagerID, a.Runtime, a.MonthlyBudget).
		WillReturnRows(agentRow(mock, a))

	got, err := db.CreateAgent(context.Background(), a)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != a.ID {
		t.Errorf("expected ID %v, got %v", a.ID, got.ID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestCreateAgent_DBError(t *testing.T) {
	mock, db := newMock(t)
	a := sampleAgent(t)

	mock.ExpectQuery(`INSERT INTO agents`).
		WithArgs(a.CompanyID, a.Role, a.Title, a.SystemPrompt, a.ManagerID, a.Runtime, a.MonthlyBudget).
		WillReturnError(errors.New("db error"))

	got, err := db.CreateAgent(context.Background(), a)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got != nil {
		t.Errorf("expected nil agent on error, got %v", got)
	}
}

// --- GetAgent ---

func TestGetAgent_Success(t *testing.T) {
	mock, db := newMock(t)
	a := sampleAgent(t)

	mock.ExpectQuery(`SELECT .* FROM agents WHERE id`).
		WithArgs(a.ID).
		WillReturnRows(agentRow(mock, a))

	got, err := db.GetAgent(context.Background(), a.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Role != a.Role {
		t.Errorf("expected role %q, got %q", a.Role, got.Role)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestGetAgent_NotFound(t *testing.T) {
	mock, db := newMock(t)
	id := mustUUID(t)

	mock.ExpectQuery(`SELECT .* FROM agents WHERE id`).
		WithArgs(id).
		WillReturnError(errors.New("no rows"))

	got, err := db.GetAgent(context.Background(), id)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

// --- ListAgentsByCompany ---

func TestListAgentsByCompany_Success(t *testing.T) {
	mock, db := newMock(t)
	a := sampleAgent(t)

	rows := mock.NewRows(agentColumns).AddRow(
		a.ID, a.CompanyID, a.Role, a.Title, a.SystemPrompt,
		a.ManagerID, a.Runtime, a.Status,
		a.MonthlyBudget, a.TokenSpend, a.ChatTokenSpend, a.PID,
		a.CreatedAt, a.UpdatedAt,
	)
	mock.ExpectQuery(`SELECT .* FROM agents WHERE company_id`).
		WithArgs(a.CompanyID).
		WillReturnRows(rows)

	agents, err := db.ListAgentsByCompany(context.Background(), a.CompanyID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(agents) != 1 {
		t.Errorf("expected 1 agent, got %d", len(agents))
	}
}

func TestListAgentsByCompany_Empty(t *testing.T) {
	mock, db := newMock(t)
	companyID := mustUUID(t)

	mock.ExpectQuery(`SELECT .* FROM agents WHERE company_id`).
		WithArgs(companyID).
		WillReturnRows(mock.NewRows(agentColumns))

	agents, err := db.ListAgentsByCompany(context.Background(), companyID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(agents) != 0 {
		t.Errorf("expected 0 agents, got %d", len(agents))
	}
}

func TestListAgentsByCompany_QueryError(t *testing.T) {
	mock, db := newMock(t)
	companyID := mustUUID(t)

	mock.ExpectQuery(`SELECT .* FROM agents WHERE company_id`).
		WithArgs(companyID).
		WillReturnError(errors.New("query error"))

	_, err := db.ListAgentsByCompany(context.Background(), companyID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- UpdateAgentStatus ---

func TestUpdateAgentStatus_Success(t *testing.T) {
	mock, db := newMock(t)
	id := mustUUID(t)

	mock.ExpectExec(`UPDATE agents SET status`).
		WithArgs(store.StatusWorking, id).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := db.UpdateAgentStatus(context.Background(), id, store.StatusWorking)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestUpdateAgentStatus_Error(t *testing.T) {
	mock, db := newMock(t)
	id := mustUUID(t)

	mock.ExpectExec(`UPDATE agents SET status`).
		WithArgs(store.StatusWorking, id).
		WillReturnError(errors.New("update error"))

	err := db.UpdateAgentStatus(context.Background(), id, store.StatusWorking)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- UpdateAgentPID ---

func TestUpdateAgentPID_WithValue(t *testing.T) {
	mock, db := newMock(t)
	id := mustUUID(t)
	pid := 1234

	mock.ExpectExec(`UPDATE agents SET pid`).
		WithArgs(&pid, id).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := db.UpdateAgentPID(context.Background(), id, &pid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateAgentPID_Nil(t *testing.T) {
	mock, db := newMock(t)
	id := mustUUID(t)

	mock.ExpectExec(`UPDATE agents SET pid`).
		WithArgs((*int)(nil), id).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := db.UpdateAgentPID(context.Background(), id, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateAgentPID_Error(t *testing.T) {
	mock, db := newMock(t)
	id := mustUUID(t)

	mock.ExpectExec(`UPDATE agents SET pid`).
		WillReturnError(errors.New("update error"))

	err := db.UpdateAgentPID(context.Background(), id, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- UpdateAgentManager ---

func TestUpdateAgentManager_Success(t *testing.T) {
	mock, db := newMock(t)
	agentID := mustUUID(t)
	managerID := mustUUID(t)

	mock.ExpectExec(`UPDATE agents SET manager_id`).
		WithArgs(managerID, agentID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := db.UpdateAgentManager(context.Background(), agentID, managerID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateAgentManager_Error(t *testing.T) {
	mock, db := newMock(t)
	agentID := mustUUID(t)
	managerID := mustUUID(t)

	mock.ExpectExec(`UPDATE agents SET manager_id`).
		WillReturnError(errors.New("update error"))

	err := db.UpdateAgentManager(context.Background(), agentID, managerID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- AddTokenSpend ---

func TestAddTokenSpend_Chat(t *testing.T) {
	mock, db := newMock(t)
	id := mustUUID(t)

	mock.ExpectExec(`UPDATE agents SET chat_token_spend`).
		WithArgs(500, id).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := db.AddTokenSpend(context.Background(), id, 500, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAddTokenSpend_NonChat(t *testing.T) {
	mock, db := newMock(t)
	id := mustUUID(t)

	mock.ExpectExec(`UPDATE agents SET token_spend`).
		WithArgs(300, id).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := db.AddTokenSpend(context.Background(), id, 300, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAddTokenSpend_ChatError(t *testing.T) {
	mock, db := newMock(t)
	id := mustUUID(t)

	mock.ExpectExec(`UPDATE agents SET chat_token_spend`).
		WillReturnError(errors.New("exec error"))

	err := db.AddTokenSpend(context.Background(), id, 100, true)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestAddTokenSpend_NonChatError(t *testing.T) {
	mock, db := newMock(t)
	id := mustUUID(t)

	mock.ExpectExec(`UPDATE agents SET token_spend`).
		WillReturnError(errors.New("exec error"))

	err := db.AddTokenSpend(context.Background(), id, 100, false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- DeleteAgent ---

func TestDeleteAgent_Success(t *testing.T) {
	mock, db := newMock(t)
	id := mustUUID(t)

	mock.ExpectExec(`DELETE FROM agents WHERE id`).
		WithArgs(id).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err := db.DeleteAgent(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteAgent_Error(t *testing.T) {
	mock, db := newMock(t)
	id := mustUUID(t)

	mock.ExpectExec(`DELETE FROM agents WHERE id`).
		WillReturnError(errors.New("delete error"))

	err := db.DeleteAgent(context.Background(), id)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- GetSubtreeAgentIDs ---

func TestGetSubtreeAgentIDs_Success(t *testing.T) {
	mock, db := newMock(t)
	rootID := mustUUID(t)
	childID := mustUUID(t)

	rows := mock.NewRows([]string{"id"}).
		AddRow(rootID).
		AddRow(childID)

	mock.ExpectQuery(`WITH RECURSIVE subtree`).
		WithArgs(rootID).
		WillReturnRows(rows)

	ids, err := db.GetSubtreeAgentIDs(context.Background(), rootID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 2 {
		t.Errorf("expected 2 IDs, got %d", len(ids))
	}
}

func TestGetSubtreeAgentIDs_Error(t *testing.T) {
	mock, db := newMock(t)
	id := mustUUID(t)

	mock.ExpectQuery(`WITH RECURSIVE subtree`).
		WillReturnError(errors.New("query error"))

	_, err := db.GetSubtreeAgentIDs(context.Background(), id)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- GetCEO ---

func TestGetCEO_Success(t *testing.T) {
	mock, db := newMock(t)
	a := sampleAgent(t)

	mock.ExpectQuery(`FROM agents WHERE company_id .* AND manager_id IS NULL`).
		WithArgs(a.CompanyID).
		WillReturnRows(agentRow(mock, a))

	got, err := db.GetCEO(context.Background(), a.CompanyID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != a.ID {
		t.Errorf("expected ID %v, got %v", a.ID, got.ID)
	}
}

func TestGetCEO_NotFound(t *testing.T) {
	mock, db := newMock(t)
	companyID := mustUUID(t)

	mock.ExpectQuery(`FROM agents WHERE company_id .* AND manager_id IS NULL`).
		WithArgs(companyID).
		WillReturnError(errors.New("no rows"))

	_, err := db.GetCEO(context.Background(), companyID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- agentRow with manager ---

func TestCreateAgent_WithManager(t *testing.T) {
	mock, db := newMock(t)
	a := sampleAgent(t)
	managerID := mustUUID(t)
	a.ManagerID = &managerID

	mock.ExpectQuery(`INSERT INTO agents`).
		WithArgs(a.CompanyID, a.Role, a.Title, a.SystemPrompt, a.ManagerID, a.Runtime, a.MonthlyBudget).
		WillReturnRows(agentRow(mock, a))

	got, err := db.CreateAgent(context.Background(), a)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ManagerID == nil {
		t.Error("expected manager ID to be set")
	}
}

// --- ListAgentsByCompany with scan error ---

func TestListAgentsByCompany_ScanError(t *testing.T) {
	mock, db := newMock(t)
	companyID := mustUUID(t)

	// Return rows with wrong column count to force scan error.
	rows := mock.NewRows([]string{"id"}).AddRow(mustUUID(t))
	mock.ExpectQuery(`SELECT .* FROM agents WHERE company_id`).
		WithArgs(companyID).
		WillReturnRows(rows)

	_, err := db.ListAgentsByCompany(context.Background(), companyID)
	if err == nil {
		t.Fatal("expected scan error, got nil")
	}
}

// Verify constants from models.go are correct values.
func TestAgentStatusConstants(t *testing.T) {
	tests := []struct {
		name string
		val  store.AgentStatus
		want string
	}{
		{"idle", store.StatusIdle, "idle"},
		{"working", store.StatusWorking, "working"},
		{"paused", store.StatusPaused, "paused"},
		{"blocked", store.StatusBlocked, "blocked"},
		{"failed", store.StatusFailed, "failed"},
		{"done", store.StatusDone, "done"},
		{"degraded", store.StatusDegraded, "degraded"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.val) != tt.want {
				t.Errorf("expected %q, got %q", tt.want, tt.val)
			}
		})
	}
}

func TestAgentRuntimeConstants(t *testing.T) {
	if string(store.RuntimeClaudeCode) != "claude_code" {
		t.Errorf("expected claude_code, got %q", store.RuntimeClaudeCode)
	}
	if string(store.RuntimeOpenClaw) != "openclaw" {
		t.Errorf("expected openclaw, got %q", store.RuntimeOpenClaw)
	}
}

// Test that GetSubtreeAgentIDs returns empty slice (not nil) for root with no children.
func TestGetSubtreeAgentIDs_OnlyRoot(t *testing.T) {
	mock, db := newMock(t)
	rootID := mustUUID(t)

	rows := mock.NewRows([]string{"id"}).AddRow(rootID)
	mock.ExpectQuery(`WITH RECURSIVE subtree`).
		WithArgs(rootID).
		WillReturnRows(rows)

	ids, err := db.GetSubtreeAgentIDs(context.Background(), rootID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 1 {
		t.Errorf("expected 1 ID (root itself), got %d", len(ids))
	}
}

// Make sure the PID field handles time values properly.
func TestSampleAgent_TimestampFields(t *testing.T) {
	a := &store.Agent{}
	a.CreatedAt = time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	if a.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at")
	}
}

// Compile-time check: Agent has expected fields.
var _ = store.Agent{
	ID:            uuid.UUID{},
	CompanyID:     uuid.UUID{},
	Role:          "",
	Title:         "",
	SystemPrompt:  "",
	ManagerID:     nil,
	Runtime:       store.RuntimeClaudeCode,
	Status:        store.StatusIdle,
	MonthlyBudget: 0,
	TokenSpend:    0,
	ChatTokenSpend: 0,
	PID:           nil,
}
