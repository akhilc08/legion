package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"conductor/internal/store"
	"github.com/google/uuid"
)

func fixedIssueID() uuid.UUID { return uuid.MustParse("dddddddd-0000-0000-0000-000000000004") }
func fixedDepID() uuid.UUID   { return uuid.MustParse("eeeeeeee-0000-0000-0000-000000000005") }

func sampleIssue(id, companyID uuid.UUID) *store.Issue {
	return &store.Issue{
		ID:          id,
		CompanyID:   companyID,
		Title:       "Fix the bug",
		Description: "It crashes",
		Status:      store.IssuePending,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// ── handleListIssues ──────────────────────────────────────────────────────────

func TestHandleListIssues_NoAuth(t *testing.T) {
	srv := newTestServer(newMockStore(), newMockOrch())
	rr := do(srv, "GET", companyPath(fixedCompanyID(), "/issues"), nil, nil)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestHandleListIssues_Empty(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	ms.listIssuesByCompanyFn = func(_ context.Context, _ uuid.UUID) ([]store.Issue, error) {
		return []store.Issue{}, nil
	}
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "GET", companyPath(fixedCompanyID(), "/issues"), nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var result []store.Issue
	decodeBodyInto(rr, &result) //nolint
	if len(result) != 0 {
		t.Fatalf("expected empty slice, got %d", len(result))
	}
}

func TestHandleListIssues_Populated(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	cid := fixedCompanyID()
	iid := fixedIssueID()
	ms.listIssuesByCompanyFn = func(_ context.Context, _ uuid.UUID) ([]store.Issue, error) {
		return []store.Issue{*sampleIssue(iid, cid)}, nil
	}
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "GET", companyPath(cid, "/issues"), nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var result []store.Issue
	decodeBodyInto(rr, &result) //nolint
	if len(result) != 1 || result[0].ID != iid {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestHandleListIssues_NilReturnedAsEmptySlice(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	ms.listIssuesByCompanyFn = func(_ context.Context, _ uuid.UUID) ([]store.Issue, error) {
		return nil, nil
	}
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "GET", companyPath(fixedCompanyID(), "/issues"), nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if body == "null\n" {
		t.Fatal("response must not be null when DB returns nil")
	}
}

func TestHandleListIssues_DBError(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	ms.listIssuesByCompanyFn = func(_ context.Context, _ uuid.UUID) ([]store.Issue, error) {
		return nil, errors.New("db error")
	}
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "GET", companyPath(fixedCompanyID(), "/issues"), nil)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

// ── handleCreateIssue ─────────────────────────────────────────────────────────

func TestHandleCreateIssue_NoAuth(t *testing.T) {
	srv := newTestServer(newMockStore(), newMockOrch())
	rr := do(srv, "POST", companyPath(fixedCompanyID(), "/issues"), map[string]string{"title": "x"}, nil)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestHandleCreateIssue_MissingTitle(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "POST", companyPath(fixedCompanyID(), "/issues"), map[string]string{"description": "no title"})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	var resp map[string]string
	decodeBodyInto(rr, &resp) //nolint
	if resp["error"] != "title is required" {
		t.Fatalf("unexpected error msg: %q", resp["error"])
	}
}

func TestHandleCreateIssue_InvalidJSON(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	srv := newTestServer(ms, newMockOrch())

	req, _ := newRawRequest("POST", companyPath(fixedCompanyID(), "/issues"), []byte("{bad"), map[string]string{
		"Authorization": authHeader(fixedUserID()),
	})
	rr := httptest.NewRecorder()
	srv.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandleCreateIssue_WithAssignee(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	mo := newMockOrch()
	cid := fixedCompanyID()
	iid := fixedIssueID()
	aid := fixedAgentID()
	assigneeTriggerred := false

	ms.createIssueFn = func(_ context.Context, issue *store.Issue) (*store.Issue, error) {
		out := *sampleIssue(iid, cid)
		out.AssigneeID = issue.AssigneeID
		return &out, nil
	}
	mo.triggerAssignFn = func(agentID uuid.UUID) {
		if agentID == aid {
			assigneeTriggerred = true
		}
	}
	srv := newTestServer(ms, mo)

	rr := doAuth(srv, fixedUserID(), "POST", companyPath(cid, "/issues"), map[string]interface{}{
		"title":       "Fix bug",
		"assignee_id": aid.String(),
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	if !assigneeTriggerred {
		t.Fatal("expected TriggerAssign to be called for assignee")
	}
}

func TestHandleCreateIssue_WithoutAssignee(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	mo := newMockOrch()
	cid := fixedCompanyID()
	iid := fixedIssueID()
	ms.createIssueFn = func(_ context.Context, _ *store.Issue) (*store.Issue, error) {
		return sampleIssue(iid, cid), nil
	}
	mo.triggerAssignCompanyFn = func(_ context.Context, _ uuid.UUID) {}
	srv := newTestServer(ms, mo)

	rr := doAuth(srv, fixedUserID(), "POST", companyPath(cid, "/issues"), map[string]interface{}{
		"title": "Unassigned task",
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	// TriggerAssignCompany runs in a goroutine so we can't guarantee it has fired yet,
	// but no error should occur.
}

func TestHandleCreateIssue_DBError(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	ms.createIssueFn = func(_ context.Context, _ *store.Issue) (*store.Issue, error) {
		return nil, errors.New("db error")
	}
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "POST", companyPath(fixedCompanyID(), "/issues"), map[string]interface{}{
		"title": "X",
	})
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestHandleCreateIssue_FKError_Returns422(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	ms.createIssueFn = func(_ context.Context, _ *store.Issue) (*store.Issue, error) {
		return nil, errors.New("violates foreign key constraint on column \"assignee_id\"")
	}
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "POST", companyPath(fixedCompanyID(), "/issues"), map[string]interface{}{
		"title": "X",
	})
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for FK violation, got %d", rr.Code)
	}
}

func TestHandleCreateIssue_ForeignKeyError_Returns422(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	ms.createIssueFn = func(_ context.Context, _ *store.Issue) (*store.Issue, error) {
		return nil, errors.New("foreign key constraint violation")
	}
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "POST", companyPath(fixedCompanyID(), "/issues"), map[string]interface{}{
		"title": "X",
	})
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", rr.Code)
	}
}

func TestHandleCreateIssue_Success(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	mo := newMockOrch()
	cid := fixedCompanyID()
	iid := fixedIssueID()

	ms.createIssueFn = func(_ context.Context, _ *store.Issue) (*store.Issue, error) {
		return sampleIssue(iid, cid), nil
	}
	mo.triggerAssignCompanyFn = func(_ context.Context, _ uuid.UUID) {}
	srv := newTestServer(ms, mo)

	rr := doAuth(srv, fixedUserID(), "POST", companyPath(cid, "/issues"), map[string]interface{}{
		"title":       "Do the thing",
		"description": "details",
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var issue store.Issue
	decodeBodyInto(rr, &issue) //nolint
	if issue.ID != iid {
		t.Fatalf("unexpected issue id %s", issue.ID)
	}
}

// ── handleGetIssue ────────────────────────────────────────────────────────────

func TestHandleGetIssue_NoAuth(t *testing.T) {
	srv := newTestServer(newMockStore(), newMockOrch())
	rr := do(srv, "GET", companyPath(fixedCompanyID(), "/issues/"+fixedIssueID().String()), nil, nil)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestHandleGetIssue_InvalidUUID(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "GET", companyPath(fixedCompanyID(), "/issues/not-a-uuid"), nil)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandleGetIssue_NotFound(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	ms.getIssueFn = func(_ context.Context, _ uuid.UUID) (*store.Issue, error) {
		return nil, errors.New("not found")
	}
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "GET", companyPath(fixedCompanyID(), "/issues/"+fixedIssueID().String()), nil)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestHandleGetIssue_Found(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	cid := fixedCompanyID()
	iid := fixedIssueID()
	ms.getIssueFn = func(_ context.Context, _ uuid.UUID) (*store.Issue, error) {
		return sampleIssue(iid, cid), nil
	}
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "GET", companyPath(cid, "/issues/"+iid.String()), nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var issue store.Issue
	decodeBodyInto(rr, &issue) //nolint
	if issue.ID != iid {
		t.Fatalf("unexpected id %s", issue.ID)
	}
}

// ── handleUpdateIssue ─────────────────────────────────────────────────────────

func TestHandleUpdateIssue_NoAuth(t *testing.T) {
	srv := newTestServer(newMockStore(), newMockOrch())
	rr := do(srv, "PATCH", companyPath(fixedCompanyID(), "/issues/"+fixedIssueID().String()), nil, nil)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestHandleUpdateIssue_InvalidUUID(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "PATCH", companyPath(fixedCompanyID(), "/issues/bad-uuid"), nil)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandleUpdateIssue_UpdateAssignee(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	iid := fixedIssueID()
	aid := fixedAgentID()
	assigneeUpdated := false

	ms.updateIssueAssigneeFn = func(_ context.Context, id, assigneeID uuid.UUID) error {
		if id == iid && assigneeID == aid {
			assigneeUpdated = true
		}
		return nil
	}
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "PATCH", companyPath(fixedCompanyID(), "/issues/"+iid.String()), map[string]interface{}{
		"assignee_id": aid.String(),
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !assigneeUpdated {
		t.Fatal("UpdateIssueAssignee was not called with correct args")
	}
}

func TestHandleUpdateIssue_UpdateStatus(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	iid := fixedIssueID()
	statusUpdated := false

	ms.updateIssueStatusFn = func(_ context.Context, id uuid.UUID, status store.IssueStatus) error {
		if id == iid && status == store.IssueDone {
			statusUpdated = true
		}
		return nil
	}
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "PATCH", companyPath(fixedCompanyID(), "/issues/"+iid.String()), map[string]interface{}{
		"status": "done",
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !statusUpdated {
		t.Fatal("UpdateIssueStatus was not called with correct args")
	}
}

func TestHandleUpdateIssue_UpdateBoth(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	iid := fixedIssueID()
	aid := fixedAgentID()
	assigneeUpdated := false
	statusUpdated := false

	ms.updateIssueAssigneeFn = func(_ context.Context, _, _ uuid.UUID) error {
		assigneeUpdated = true
		return nil
	}
	ms.updateIssueStatusFn = func(_ context.Context, _ uuid.UUID, _ store.IssueStatus) error {
		statusUpdated = true
		return nil
	}
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "PATCH", companyPath(fixedCompanyID(), "/issues/"+iid.String()), map[string]interface{}{
		"assignee_id": aid.String(),
		"status":      "in_progress",
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !assigneeUpdated || !statusUpdated {
		t.Fatalf("both updates should be called: assignee=%v status=%v", assigneeUpdated, statusUpdated)
	}
}

func TestHandleUpdateIssue_DBError(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	ms.updateIssueAssigneeFn = func(_ context.Context, _, _ uuid.UUID) error {
		return errors.New("db error")
	}
	srv := newTestServer(ms, newMockOrch())

	aid := uuid.New()
	rr := doAuth(srv, fixedUserID(), "PATCH", companyPath(fixedCompanyID(), "/issues/"+fixedIssueID().String()), map[string]interface{}{
		"assignee_id": aid.String(),
	})
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestHandleUpdateIssue_StatusDBError(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	ms.updateIssueStatusFn = func(_ context.Context, _ uuid.UUID, _ store.IssueStatus) error {
		return errors.New("db error")
	}
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "PATCH", companyPath(fixedCompanyID(), "/issues/"+fixedIssueID().String()), map[string]interface{}{
		"status": "done",
	})
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestHandleUpdateIssue_NoFieldsIsOK(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	srv := newTestServer(ms, newMockOrch())

	// PATCH with empty body — no fields to update, should still return 200
	rr := doAuth(srv, fixedUserID(), "PATCH", companyPath(fixedCompanyID(), "/issues/"+fixedIssueID().String()), map[string]interface{}{})
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

// ── handleAddDependency ───────────────────────────────────────────────────────

func TestHandleAddDependency_NoAuth(t *testing.T) {
	srv := newTestServer(newMockStore(), newMockOrch())
	rr := do(srv, "POST", companyPath(fixedCompanyID(), "/issues/"+fixedIssueID().String()+"/dependencies"), nil, nil)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestHandleAddDependency_InvalidIssueUUID(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "POST", companyPath(fixedCompanyID(), "/issues/bad-uuid/dependencies"), map[string]string{
		"depends_on_id": uuid.New().String(),
	})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandleAddDependency_InvalidJSON(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	srv := newTestServer(ms, newMockOrch())

	req, _ := newRawRequest("POST", companyPath(fixedCompanyID(), "/issues/"+fixedIssueID().String()+"/dependencies"), []byte("{bad"), map[string]string{
		"Authorization": authHeader(fixedUserID()),
	})
	rr := httptest.NewRecorder()
	srv.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandleAddDependency_DBError(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	ms.addDependencyFn = func(_ context.Context, _, _ uuid.UUID) error {
		return errors.New("cycle detected")
	}
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "POST", companyPath(fixedCompanyID(), "/issues/"+fixedIssueID().String()+"/dependencies"), map[string]interface{}{
		"depends_on_id": fixedDepID().String(),
	})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandleAddDependency_Success(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	ms.addDependencyFn = func(_ context.Context, _, _ uuid.UUID) error { return nil }
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "POST", companyPath(fixedCompanyID(), "/issues/"+fixedIssueID().String()+"/dependencies"), map[string]interface{}{
		"depends_on_id": fixedDepID().String(),
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

// ── handleRemoveDependency ────────────────────────────────────────────────────

func TestHandleRemoveDependency_NoAuth(t *testing.T) {
	srv := newTestServer(newMockStore(), newMockOrch())
	rr := do(srv, "DELETE", companyPath(fixedCompanyID(), "/issues/"+fixedIssueID().String()+"/dependencies/"+fixedDepID().String()), nil, nil)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestHandleRemoveDependency_InvalidIssueUUID(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "DELETE", companyPath(fixedCompanyID(), "/issues/bad-uuid/dependencies/"+fixedDepID().String()), nil)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandleRemoveDependency_InvalidDepUUID(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "DELETE", companyPath(fixedCompanyID(), "/issues/"+fixedIssueID().String()+"/dependencies/bad-dep-uuid"), nil)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandleRemoveDependency_DBError(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	ms.removeDependencyFn = func(_ context.Context, _, _ uuid.UUID) error {
		return errors.New("db error")
	}
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "DELETE", companyPath(fixedCompanyID(), "/issues/"+fixedIssueID().String()+"/dependencies/"+fixedDepID().String()), nil)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestHandleRemoveDependency_Success(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	ms.removeDependencyFn = func(_ context.Context, _, _ uuid.UUID) error { return nil }
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "DELETE", companyPath(fixedCompanyID(), "/issues/"+fixedIssueID().String()+"/dependencies/"+fixedDepID().String()), nil)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}
}
