package store_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/pashagolub/pgxmock/v2"

	"conductor/internal/store"
)

var issueColumns = []string{
	"id", "company_id", "title", "description", "assignee_id", "parent_id",
	"status", "output_path", "created_by", "attempt_count", "last_failure_reason",
	"escalation_id", "created_at", "updated_at",
}

func sampleIssue(t *testing.T) *store.Issue {
	t.Helper()
	return &store.Issue{
		ID:          mustUUID(t),
		CompanyID:   mustUUID(t),
		Title:       "Fix bug",
		Description: "There is a bug",
		Status:      store.IssuePending,
		CreatedAt:   fixedTime(),
		UpdatedAt:   fixedTime(),
	}
}

func issueRow(mock pgxmock.PgxPoolIface, i *store.Issue) *pgxmock.Rows {
	return mock.NewRows(issueColumns).AddRow(
		i.ID, i.CompanyID, i.Title, i.Description, i.AssigneeID, i.ParentID,
		i.Status, i.OutputPath, i.CreatedBy, i.AttemptCount, i.LastFailureReason,
		i.EscalationID, i.CreatedAt, i.UpdatedAt,
	)
}

// --- CreateIssue ---

func TestCreateIssue_Success(t *testing.T) {
	mock, db := newMock(t)
	issue := sampleIssue(t)

	mock.ExpectQuery(`INSERT INTO issues`).
		WithArgs(issue.CompanyID, issue.Title, issue.Description, issue.AssigneeID, issue.ParentID, issue.CreatedBy).
		WillReturnRows(issueRow(mock, issue))

	got, err := db.CreateIssue(context.Background(), issue)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != issue.ID {
		t.Errorf("expected ID %v, got %v", issue.ID, got.ID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestCreateIssue_Error(t *testing.T) {
	mock, db := newMock(t)
	issue := sampleIssue(t)

	mock.ExpectQuery(`INSERT INTO issues`).
		WillReturnError(errors.New("db error"))

	got, err := db.CreateIssue(context.Background(), issue)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got != nil {
		t.Error("expected nil issue on error")
	}
}

func TestCreateIssue_WithOptionalFields(t *testing.T) {
	mock, db := newMock(t)
	issue := sampleIssue(t)
	assigneeID := mustUUID(t)
	parentID := mustUUID(t)
	createdBy := mustUUID(t)
	issue.AssigneeID = &assigneeID
	issue.ParentID = &parentID
	issue.CreatedBy = &createdBy

	mock.ExpectQuery(`INSERT INTO issues`).
		WithArgs(issue.CompanyID, issue.Title, issue.Description, issue.AssigneeID, issue.ParentID, issue.CreatedBy).
		WillReturnRows(issueRow(mock, issue))

	got, err := db.CreateIssue(context.Background(), issue)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.AssigneeID == nil {
		t.Error("expected assignee_id to be set")
	}
}

// --- GetIssue ---

func TestGetIssue_Success(t *testing.T) {
	mock, db := newMock(t)
	issue := sampleIssue(t)

	mock.ExpectQuery(`SELECT .* FROM issues WHERE id`).
		WithArgs(issue.ID).
		WillReturnRows(issueRow(mock, issue))

	got, err := db.GetIssue(context.Background(), issue.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Title != issue.Title {
		t.Errorf("expected title %q, got %q", issue.Title, got.Title)
	}
}

func TestGetIssue_NotFound(t *testing.T) {
	mock, db := newMock(t)
	id := mustUUID(t)

	mock.ExpectQuery(`SELECT .* FROM issues WHERE id`).
		WithArgs(id).
		WillReturnError(errors.New("no rows"))

	got, err := db.GetIssue(context.Background(), id)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got != nil {
		t.Error("expected nil issue")
	}
}

// --- ListIssuesByCompany ---

func TestListIssuesByCompany_Success(t *testing.T) {
	mock, db := newMock(t)
	issue := sampleIssue(t)

	mock.ExpectQuery(`SELECT .* FROM issues WHERE company_id`).
		WithArgs(issue.CompanyID).
		WillReturnRows(issueRow(mock, issue))

	issues, err := db.ListIssuesByCompany(context.Background(), issue.CompanyID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 1 {
		t.Errorf("expected 1 issue, got %d", len(issues))
	}
}

func TestListIssuesByCompany_Empty(t *testing.T) {
	mock, db := newMock(t)
	companyID := mustUUID(t)

	mock.ExpectQuery(`SELECT .* FROM issues WHERE company_id`).
		WithArgs(companyID).
		WillReturnRows(mock.NewRows(issueColumns))

	issues, err := db.ListIssuesByCompany(context.Background(), companyID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 0 {
		t.Errorf("expected 0 issues, got %d", len(issues))
	}
}

func TestListIssuesByCompany_Error(t *testing.T) {
	mock, db := newMock(t)
	companyID := mustUUID(t)

	mock.ExpectQuery(`SELECT .* FROM issues WHERE company_id`).
		WillReturnError(errors.New("query error"))

	_, err := db.ListIssuesByCompany(context.Background(), companyID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- ListReadyIssues ---

func TestListReadyIssues_Success(t *testing.T) {
	mock, db := newMock(t)
	issue := sampleIssue(t)

	mock.ExpectQuery(`SELECT i\.id, i\.company_id`).
		WithArgs(issue.CompanyID).
		WillReturnRows(issueRow(mock, issue))

	issues, err := db.ListReadyIssues(context.Background(), issue.CompanyID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 1 {
		t.Errorf("expected 1 issue, got %d", len(issues))
	}
}

func TestListReadyIssues_Empty(t *testing.T) {
	mock, db := newMock(t)
	companyID := mustUUID(t)

	mock.ExpectQuery(`SELECT i\.id, i\.company_id`).
		WithArgs(companyID).
		WillReturnRows(mock.NewRows(issueColumns))

	issues, err := db.ListReadyIssues(context.Background(), companyID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 0 {
		t.Errorf("expected 0 issues, got %d", len(issues))
	}
}

func TestListReadyIssues_Error(t *testing.T) {
	mock, db := newMock(t)
	companyID := mustUUID(t)

	mock.ExpectQuery(`SELECT i\.id, i\.company_id`).
		WillReturnError(errors.New("query error"))

	_, err := db.ListReadyIssues(context.Background(), companyID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- UpdateIssueStatus ---

func TestUpdateIssueStatus_Success(t *testing.T) {
	mock, db := newMock(t)
	id := mustUUID(t)

	mock.ExpectExec(`UPDATE issues SET status`).
		WithArgs(store.IssueInProgress, id).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := db.UpdateIssueStatus(context.Background(), id, store.IssueInProgress)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateIssueStatus_Error(t *testing.T) {
	mock, db := newMock(t)
	id := mustUUID(t)

	mock.ExpectExec(`UPDATE issues SET status`).
		WillReturnError(errors.New("update error"))

	err := db.UpdateIssueStatus(context.Background(), id, store.IssueDone)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- UpdateIssueAssignee ---

func TestUpdateIssueAssignee_Success(t *testing.T) {
	mock, db := newMock(t)
	id := mustUUID(t)
	assigneeID := mustUUID(t)

	mock.ExpectExec(`UPDATE issues SET assignee_id`).
		WithArgs(assigneeID, id).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := db.UpdateIssueAssignee(context.Background(), id, assigneeID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateIssueAssignee_Error(t *testing.T) {
	mock, db := newMock(t)

	mock.ExpectExec(`UPDATE issues SET assignee_id`).
		WillReturnError(errors.New("update error"))

	err := db.UpdateIssueAssignee(context.Background(), mustUUID(t), mustUUID(t))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- UpdateIssueOutput ---

func TestUpdateIssueOutput_Success(t *testing.T) {
	mock, db := newMock(t)
	id := mustUUID(t)

	mock.ExpectExec(`UPDATE issues SET output_path`).
		WithArgs("/output/result.txt", id).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := db.UpdateIssueOutput(context.Background(), id, "/output/result.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateIssueOutput_Error(t *testing.T) {
	mock, db := newMock(t)

	mock.ExpectExec(`UPDATE issues SET output_path`).
		WillReturnError(errors.New("update error"))

	err := db.UpdateIssueOutput(context.Background(), mustUUID(t), "/out")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- IncrementAttemptCount ---

func TestIncrementAttemptCount_Success(t *testing.T) {
	mock, db := newMock(t)
	id := mustUUID(t)

	mock.ExpectExec(`UPDATE issues SET attempt_count`).
		WithArgs("timeout", id).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := db.IncrementAttemptCount(context.Background(), id, "timeout")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIncrementAttemptCount_Error(t *testing.T) {
	mock, db := newMock(t)

	mock.ExpectExec(`UPDATE issues SET attempt_count`).
		WillReturnError(errors.New("update error"))

	err := db.IncrementAttemptCount(context.Background(), mustUUID(t), "reason")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- CheckoutIssue ---

func TestCheckoutIssue_Success(t *testing.T) {
	mock, db := newMock(t)
	issueID := mustUUID(t)
	agentID := mustUUID(t)
	lockKey := int64(issueID[0])<<24 | int64(issueID[1])<<16 | int64(issueID[2])<<8 | int64(issueID[3])

	mock.ExpectQuery(`SELECT pg_try_advisory_lock`).
		WithArgs(lockKey).
		WillReturnRows(mock.NewRows([]string{"pg_try_advisory_lock"}).AddRow(true))

	mock.ExpectExec(`UPDATE issues SET status = 'in_progress'`).
		WithArgs(agentID, issueID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	checked, err := db.CheckoutIssue(context.Background(), issueID, agentID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !checked {
		t.Error("expected checked=true")
	}
}

func TestCheckoutIssue_AlreadyLocked(t *testing.T) {
	mock, db := newMock(t)
	issueID := mustUUID(t)
	agentID := mustUUID(t)
	lockKey := int64(issueID[0])<<24 | int64(issueID[1])<<16 | int64(issueID[2])<<8 | int64(issueID[3])

	mock.ExpectQuery(`SELECT pg_try_advisory_lock`).
		WithArgs(lockKey).
		WillReturnRows(mock.NewRows([]string{"pg_try_advisory_lock"}).AddRow(false))

	checked, err := db.CheckoutIssue(context.Background(), issueID, agentID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if checked {
		t.Error("expected checked=false when lock not acquired")
	}
}

func TestCheckoutIssue_LockError(t *testing.T) {
	mock, db := newMock(t)
	issueID := mustUUID(t)
	agentID := mustUUID(t)
	lockKey := int64(issueID[0])<<24 | int64(issueID[1])<<16 | int64(issueID[2])<<8 | int64(issueID[3])

	mock.ExpectQuery(`SELECT pg_try_advisory_lock`).
		WithArgs(lockKey).
		WillReturnError(errors.New("lock error"))

	_, err := db.CheckoutIssue(context.Background(), issueID, agentID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCheckoutIssue_RowsAffectedZero(t *testing.T) {
	// Locked but issue was already taken (status != pending).
	mock, db := newMock(t)
	issueID := mustUUID(t)
	agentID := mustUUID(t)
	lockKey := int64(issueID[0])<<24 | int64(issueID[1])<<16 | int64(issueID[2])<<8 | int64(issueID[3])

	mock.ExpectQuery(`SELECT pg_try_advisory_lock`).
		WithArgs(lockKey).
		WillReturnRows(mock.NewRows([]string{"pg_try_advisory_lock"}).AddRow(true))

	mock.ExpectExec(`UPDATE issues SET status = 'in_progress'`).
		WithArgs(agentID, issueID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	// The advisory unlock is also fired.
	mock.ExpectExec(`SELECT pg_advisory_unlock`).
		WithArgs(lockKey).
		WillReturnResult(pgxmock.NewResult("SELECT", 1))

	checked, err := db.CheckoutIssue(context.Background(), issueID, agentID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if checked {
		t.Error("expected checked=false when 0 rows affected")
	}
}

// --- AddDependency ---

func TestAddDependency_Success(t *testing.T) {
	mock, db := newMock(t)
	issueID := mustUUID(t)
	dependsOnID := mustUUID(t)

	// Cycle check query.
	mock.ExpectQuery(`WITH RECURSIVE deps AS`).
		WithArgs(dependsOnID, issueID).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(false))

	// Insert dependency.
	mock.ExpectExec(`INSERT INTO issue_dependencies`).
		WithArgs(issueID, dependsOnID).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err := db.AddDependency(context.Background(), issueID, dependsOnID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestAddDependency_CycleDetected(t *testing.T) {
	mock, db := newMock(t)
	issueID := mustUUID(t)
	dependsOnID := mustUUID(t)

	mock.ExpectQuery(`WITH RECURSIVE deps AS`).
		WithArgs(dependsOnID, issueID).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))

	err := db.AddDependency(context.Background(), issueID, dependsOnID)
	if err == nil {
		t.Fatal("expected cycle error, got nil")
	}
	if err.Error() != "dependency would create a cycle" {
		t.Errorf("expected cycle error message, got %q", err.Error())
	}
}

func TestAddDependency_CycleCheckError(t *testing.T) {
	mock, db := newMock(t)
	issueID := mustUUID(t)
	dependsOnID := mustUUID(t)

	mock.ExpectQuery(`WITH RECURSIVE deps AS`).
		WillReturnError(errors.New("query error"))

	err := db.AddDependency(context.Background(), issueID, dependsOnID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestAddDependency_InsertError(t *testing.T) {
	mock, db := newMock(t)
	issueID := mustUUID(t)
	dependsOnID := mustUUID(t)

	mock.ExpectQuery(`WITH RECURSIVE deps AS`).
		WithArgs(dependsOnID, issueID).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(false))

	mock.ExpectExec(`INSERT INTO issue_dependencies`).
		WillReturnError(errors.New("insert error"))

	err := db.AddDependency(context.Background(), issueID, dependsOnID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- RemoveDependency ---

func TestRemoveDependency_Success(t *testing.T) {
	mock, db := newMock(t)
	issueID := mustUUID(t)
	dependsOnID := mustUUID(t)

	mock.ExpectExec(`DELETE FROM issue_dependencies`).
		WithArgs(issueID, dependsOnID).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err := db.RemoveDependency(context.Background(), issueID, dependsOnID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRemoveDependency_Error(t *testing.T) {
	mock, db := newMock(t)

	mock.ExpectExec(`DELETE FROM issue_dependencies`).
		WillReturnError(errors.New("delete error"))

	err := db.RemoveDependency(context.Background(), mustUUID(t), mustUUID(t))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- GetDependencies ---

func TestGetDependencies_Success(t *testing.T) {
	mock, db := newMock(t)
	issueID := mustUUID(t)
	dep1 := mustUUID(t)
	dep2 := mustUUID(t)

	mock.ExpectQuery(`SELECT depends_on_issue_id FROM issue_dependencies`).
		WithArgs(issueID).
		WillReturnRows(mock.NewRows([]string{"depends_on_issue_id"}).
			AddRow(dep1).AddRow(dep2))

	deps, err := db.GetDependencies(context.Background(), issueID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deps) != 2 {
		t.Errorf("expected 2 dependencies, got %d", len(deps))
	}
}

func TestGetDependencies_Empty(t *testing.T) {
	mock, db := newMock(t)
	issueID := mustUUID(t)

	mock.ExpectQuery(`SELECT depends_on_issue_id FROM issue_dependencies`).
		WithArgs(issueID).
		WillReturnRows(mock.NewRows([]string{"depends_on_issue_id"}))

	deps, err := db.GetDependencies(context.Background(), issueID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deps) != 0 {
		t.Errorf("expected 0 dependencies, got %d", len(deps))
	}
}

func TestGetDependencies_Error(t *testing.T) {
	mock, db := newMock(t)
	issueID := mustUUID(t)

	mock.ExpectQuery(`SELECT depends_on_issue_id FROM issue_dependencies`).
		WillReturnError(errors.New("query error"))

	_, err := db.GetDependencies(context.Background(), issueID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// Verify IssueStatus constants.
func TestIssueStatusConstants(t *testing.T) {
	cases := []struct {
		val  store.IssueStatus
		want string
	}{
		{store.IssuePending, "pending"},
		{store.IssueInProgress, "in_progress"},
		{store.IssueBlocked, "blocked"},
		{store.IssueDone, "done"},
		{store.IssueFailed, "failed"},
	}
	for _, c := range cases {
		if string(c.val) != c.want {
			t.Errorf("expected %q, got %q", c.want, c.val)
		}
	}
}

// Compile-time check on Issue struct fields.
var _ = store.Issue{
	ID:                uuid.UUID{},
	CompanyID:         uuid.UUID{},
	Title:             "",
	Description:       "",
	AssigneeID:        nil,
	ParentID:          nil,
	Status:            store.IssuePending,
	OutputPath:        nil,
	CreatedBy:         nil,
	AttemptCount:      0,
	LastFailureReason: nil,
	EscalationID:      nil,
}
