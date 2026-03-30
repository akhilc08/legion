package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"conductor/internal/store"
	"github.com/google/uuid"
)

// ── helpers ──────────────────────────────────────────────────────────────────

func fixedUserID() uuid.UUID  { return uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000001") }
func fixedCompanyID() uuid.UUID { return uuid.MustParse("bbbbbbbb-0000-0000-0000-000000000002") }
func fixedAgentID() uuid.UUID  { return uuid.MustParse("cccccccc-0000-0000-0000-000000000003") }

// sampleCompany returns a minimal store.Company with the given id.
func sampleCompany(id uuid.UUID) *store.Company {
	return &store.Company{ID: id, Name: "Acme", Goal: "World domination", CreatedAt: time.Now()}
}

// sampleAgent returns a minimal store.Agent.
func sampleAgent(id, companyID uuid.UUID) *store.Agent {
	return &store.Agent{
		ID:            id,
		CompanyID:     companyID,
		Role:          "engineer",
		Title:         "Engineer",
		Runtime:       store.RuntimeClaudeCode,
		Status:        store.StatusIdle,
		MonthlyBudget: 100000,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

// authCompanyServer returns a server whose store always grants company access.
func authCompanyServer(ms *mockStore, mo *mockOrch) *Server {
	setupCompanyAccess(ms)
	return newTestServer(ms, mo)
}

// ── handleListCompanies ───────────────────────────────────────────────────────

func TestHandleListCompanies_NoAuth(t *testing.T) {
	ms := newMockStore()
	srv := newTestServer(ms, newMockOrch())

	rr := do(srv, "GET", "/api/companies", nil, nil)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestHandleListCompanies_EmptyList(t *testing.T) {
	ms := newMockStore()
	ms.listCompaniesForUserFn = func(_ context.Context, _ uuid.UUID) ([]store.Company, error) {
		return []store.Company{}, nil
	}
	srv := newTestServer(ms, newMockOrch())
	userID := fixedUserID()

	rr := doAuth(srv, userID, "GET", "/api/companies", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var result []store.Company
	if err := decodeBodyInto(rr, &result); err != nil {
		t.Fatal(err)
	}
	if len(result) != 0 {
		t.Fatalf("expected empty slice, got %d items", len(result))
	}
}

func TestHandleListCompanies_WithData(t *testing.T) {
	ms := newMockStore()
	cid := fixedCompanyID()
	ms.listCompaniesForUserFn = func(_ context.Context, _ uuid.UUID) ([]store.Company, error) {
		return []store.Company{*sampleCompany(cid)}, nil
	}
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "GET", "/api/companies", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var result []store.Company
	decodeBodyInto(rr, &result) //nolint
	if len(result) != 1 {
		t.Fatalf("expected 1 company, got %d", len(result))
	}
	if result[0].ID != cid {
		t.Fatalf("unexpected company id %s", result[0].ID)
	}
}

func TestHandleListCompanies_NilReturnedAsEmptySlice(t *testing.T) {
	ms := newMockStore()
	ms.listCompaniesForUserFn = func(_ context.Context, _ uuid.UUID) ([]store.Company, error) {
		return nil, nil // DB returns nil slice
	}
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "GET", "/api/companies", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	// Response must be [] not null
	if rr.Body.String() == "null\n" {
		t.Fatal("response body must not be null")
	}
	var result []interface{}
	decodeBodyInto(rr, &result) //nolint
	if result == nil {
		t.Fatal("expected non-nil decoded slice")
	}
}

func TestHandleListCompanies_DBError(t *testing.T) {
	ms := newMockStore()
	ms.listCompaniesForUserFn = func(_ context.Context, _ uuid.UUID) ([]store.Company, error) {
		return nil, errors.New("db gone")
	}
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "GET", "/api/companies", nil)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

// ── handleCreateCompany ───────────────────────────────────────────────────────

func TestHandleCreateCompany_NoAuth(t *testing.T) {
	srv := newTestServer(newMockStore(), newMockOrch())
	rr := do(srv, "POST", "/api/companies", map[string]string{"name": "X"}, nil)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestHandleCreateCompany_InvalidJSON(t *testing.T) {
	// Rebuild with raw body
	req, _ := newRawRequest("POST", "/api/companies", []byte("{bad json"), map[string]string{
		"Authorization": authHeader(fixedUserID()),
	})
	rr2 := httptest.NewRecorder()
	ms := newMockStore()
	ms.createCompanyFn = func(_ context.Context, _, _ string) (*store.Company, error) {
		return sampleCompany(fixedCompanyID()), nil
	}
	ms.addUserToCompanyFn = func(_ context.Context, _, _ uuid.UUID, _ string) error { return nil }
	ms.createAgentFn = func(_ context.Context, _ *store.Agent) (*store.Agent, error) {
		return sampleAgent(fixedAgentID(), fixedCompanyID()), nil
	}
	newTestServer(ms, newMockOrch()).Router().ServeHTTP(rr2, req)
	if rr2.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad JSON, got %d", rr2.Code)
	}
}

func TestHandleCreateCompany_Success(t *testing.T) {
	ms := newMockStore()
	cid := fixedCompanyID()
	aid := fixedAgentID()
	agentCreated := false

	ms.createCompanyFn = func(_ context.Context, name, goal string) (*store.Company, error) {
		return &store.Company{ID: cid, Name: name, Goal: goal, CreatedAt: time.Now()}, nil
	}
	ms.addUserToCompanyFn = func(_ context.Context, _, _ uuid.UUID, _ string) error { return nil }
	ms.createAgentFn = func(_ context.Context, a *store.Agent) (*store.Agent, error) {
		agentCreated = true
		return sampleAgent(aid, cid), nil
	}

	srv := newTestServer(ms, newMockOrch())
	rr := doAuth(srv, fixedUserID(), "POST", "/api/companies", map[string]string{"name": "Acme", "goal": "win"})

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	if !agentCreated {
		t.Fatal("expected Board agent to be seeded via CreateAgent")
	}
	var c store.Company
	decodeBodyInto(rr, &c) //nolint
	if c.ID != cid {
		t.Fatalf("unexpected company id %s", c.ID)
	}
}

func TestHandleCreateCompany_DBError(t *testing.T) {
	ms := newMockStore()
	ms.createCompanyFn = func(_ context.Context, _, _ string) (*store.Company, error) {
		return nil, errors.New("db error")
	}
	srv := newTestServer(ms, newMockOrch())
	rr := doAuth(srv, fixedUserID(), "POST", "/api/companies", map[string]string{"name": "X"})
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestHandleCreateCompany_BoardAgentSeeded(t *testing.T) {
	ms := newMockStore()
	cid := fixedCompanyID()
	var capturedAgent *store.Agent

	ms.createCompanyFn = func(_ context.Context, _, _ string) (*store.Company, error) {
		return sampleCompany(cid), nil
	}
	ms.addUserToCompanyFn = func(_ context.Context, _, _ uuid.UUID, _ string) error { return nil }
	ms.createAgentFn = func(_ context.Context, a *store.Agent) (*store.Agent, error) {
		capturedAgent = a
		return sampleAgent(fixedAgentID(), cid), nil
	}

	srv := newTestServer(ms, newMockOrch())
	doAuth(srv, fixedUserID(), "POST", "/api/companies", map[string]string{"name": "X"})

	if capturedAgent == nil {
		t.Fatal("CreateAgent was not called")
	}
	if capturedAgent.Role != "board" {
		t.Fatalf("expected board agent role, got %q", capturedAgent.Role)
	}
	if capturedAgent.CompanyID != cid {
		t.Fatalf("board agent company mismatch: %s vs %s", capturedAgent.CompanyID, cid)
	}
}

// ── handleGetCompany ──────────────────────────────────────────────────────────

func TestHandleGetCompany_NoAuth(t *testing.T) {
	srv := newTestServer(newMockStore(), newMockOrch())
	rr := do(srv, "GET", companyPath(fixedCompanyID(), "/"), nil, nil)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestHandleGetCompany_InvalidUUID(t *testing.T) {
	ms := newMockStore()
	ms.userCanAccessFn = func(_ context.Context, _, _ uuid.UUID) (bool, error) { return true, nil }
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "GET", "/api/companies/not-a-uuid/", nil)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandleGetCompany_NotFound(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	ms.getCompanyFn = func(_ context.Context, _ uuid.UUID) (*store.Company, error) {
		return nil, errors.New("not found")
	}
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "GET", companyPath(fixedCompanyID(), "/"), nil)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestHandleGetCompany_Found(t *testing.T) {
	ms := newMockStore()
	cid := fixedCompanyID()
	setupCompanyAccess(ms)
	ms.getCompanyFn = func(_ context.Context, _ uuid.UUID) (*store.Company, error) {
		return sampleCompany(cid), nil
	}
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "GET", companyPath(cid, "/"), nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var c store.Company
	decodeBodyInto(rr, &c) //nolint
	if c.ID != cid {
		t.Fatalf("unexpected id %s", c.ID)
	}
}

// ── handleUpdateCompany ───────────────────────────────────────────────────────

func TestHandleUpdateCompany_NoAuth(t *testing.T) {
	srv := newTestServer(newMockStore(), newMockOrch())
	rr := do(srv, "PATCH", companyPath(fixedCompanyID(), "/"), map[string]string{"goal": "x"}, nil)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestHandleUpdateCompany_InvalidJSON(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	srv := newTestServer(ms, newMockOrch())

	req, _ := newRawRequest("PATCH", companyPath(fixedCompanyID(), "/"), []byte("{bad"), map[string]string{
		"Authorization": authHeader(fixedUserID()),
	})
	rr := httptest.NewRecorder()
	srv.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandleUpdateCompany_DBError(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	ms.updateCompanyGoalFn = func(_ context.Context, _ uuid.UUID, _ string) error {
		return errors.New("db error")
	}
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "PATCH", companyPath(fixedCompanyID(), "/"), map[string]string{"goal": "new goal"})
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestHandleUpdateCompany_Success(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	ms.updateCompanyGoalFn = func(_ context.Context, _ uuid.UUID, _ string) error { return nil }
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "PATCH", companyPath(fixedCompanyID(), "/"), map[string]string{"goal": "new goal"})
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var resp map[string]string
	decodeBodyInto(rr, &resp) //nolint
	if resp["status"] != "ok" {
		t.Fatalf("expected status ok, got %q", resp["status"])
	}
}

// ── handleDeleteCompany ───────────────────────────────────────────────────────

func TestHandleDeleteCompany_NoAuth(t *testing.T) {
	srv := newTestServer(newMockStore(), newMockOrch())
	rr := do(srv, "DELETE", companyPath(fixedCompanyID(), "/"), nil, nil)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestHandleDeleteCompany_DBError(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	ms.deleteCompanyFn = func(_ context.Context, _ uuid.UUID) error { return errors.New("db error") }
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "DELETE", companyPath(fixedCompanyID(), "/"), nil)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestHandleDeleteCompany_Success(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	ms.deleteCompanyFn = func(_ context.Context, _ uuid.UUID) error { return nil }
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "DELETE", companyPath(fixedCompanyID(), "/"), nil)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}
}

// ── handleSetGoal ─────────────────────────────────────────────────────────────

func TestHandleSetGoal_NoAuth(t *testing.T) {
	srv := newTestServer(newMockStore(), newMockOrch())
	rr := do(srv, "POST", companyPath(fixedCompanyID(), "/goal"), map[string]string{"goal": "x"}, nil)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestHandleSetGoal_InvalidJSON(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	srv := newTestServer(ms, newMockOrch())

	req, _ := newRawRequest("POST", companyPath(fixedCompanyID(), "/goal"), []byte("{bad"), map[string]string{
		"Authorization": authHeader(fixedUserID()),
	})
	rr := httptest.NewRecorder()
	srv.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandleSetGoal_DBError(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	ms.updateCompanyGoalFn = func(_ context.Context, _ uuid.UUID, _ string) error {
		return errors.New("db error")
	}
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "POST", companyPath(fixedCompanyID(), "/goal"), map[string]string{"goal": "x"})
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestHandleSetGoal_Success(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	ms.updateCompanyGoalFn = func(_ context.Context, _ uuid.UUID, _ string) error { return nil }
	ms.getCEOFn = func(_ context.Context, _ uuid.UUID) (*store.Agent, error) {
		return nil, errors.New("no CEO")
	}
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "POST", companyPath(fixedCompanyID(), "/goal"), map[string]string{"goal": "x"})
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]string
	decodeBodyInto(rr, &resp) //nolint
	if resp["status"] != "ok" {
		t.Fatalf("expected status ok, got %q", resp["status"])
	}
}

func TestHandleSetGoal_WithCEO_CreatesIssue(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	cid := fixedCompanyID()
	ceoID := fixedAgentID()
	issueCreated := false

	ms.updateCompanyGoalFn = func(_ context.Context, _ uuid.UUID, _ string) error { return nil }
	ms.getCEOFn = func(_ context.Context, _ uuid.UUID) (*store.Agent, error) {
		return sampleAgent(ceoID, cid), nil
	}
	ms.createIssueFn = func(_ context.Context, issue *store.Issue) (*store.Issue, error) {
		issueCreated = true
		if issue.AssigneeID == nil || *issue.AssigneeID != ceoID {
			return nil, errors.New("wrong assignee")
		}
		return &store.Issue{
			ID:          uuid.New(),
			CompanyID:   cid,
			Title:       issue.Title,
			Description: issue.Description,
			AssigneeID:  issue.AssigneeID,
			Status:      store.IssuePending,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}, nil
	}
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "POST", companyPath(cid, "/goal"), map[string]string{"goal": "become #1"})
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !issueCreated {
		t.Fatal("expected CreateIssue to be called when CEO exists")
	}
}

func TestHandleSetGoal_NoCEO_NoIssueCreated(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	issueCreated := false

	ms.updateCompanyGoalFn = func(_ context.Context, _ uuid.UUID, _ string) error { return nil }
	ms.getCEOFn = func(_ context.Context, _ uuid.UUID) (*store.Agent, error) {
		return nil, errors.New("no CEO")
	}
	ms.createIssueFn = func(_ context.Context, _ *store.Issue) (*store.Issue, error) {
		issueCreated = true
		return nil, nil
	}
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "POST", companyPath(fixedCompanyID(), "/goal"), map[string]string{"goal": "x"})
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if issueCreated {
		t.Fatal("CreateIssue should NOT be called when no CEO exists")
	}
}

// ── utility ───────────────────────────────────────────────────────────────────

// newRawRequest creates an http.Request with a raw body (for invalid JSON tests).
func newRawRequest(method, path string, body []byte, headers map[string]string) (*http.Request, error) {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return req, nil
}

// httptest is needed for NewRecorder in the test file
var _ = json.Marshal // keep json import used
