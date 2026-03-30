package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"conductor/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// ── test-server helpers ───────────────────────────────────────────────────────

func newHiringTestServer(db *mockStore, orch *mockOrch) *Server {
	s := &Server{
		db:        db,
		orch:      orch,
		jwtSecret: "test-secret",
	}
	return s
}

// reqWithAuth attaches a userID and companyID to the request context,
// simulating what authMiddleware + chiCompanyMiddleware would do.
func reqWithAuth(method, target string, body []byte, userID, companyID uuid.UUID) *http.Request {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, target, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, target, nil)
	}
	ctx := context.WithValue(req.Context(), ctxUserID, userID)
	ctx = context.WithValue(ctx, ctxCompanyID, companyID)
	return req.WithContext(ctx)
}

// reqNoAuth returns a request with no auth context set.
func reqNoAuth(method, target string, body []byte) *http.Request {
	if body != nil {
		req := httptest.NewRequest(method, target, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		return req
	}
	return httptest.NewRequest(method, target, nil)
}

// chiParam wraps a handler so that chi.URLParam(r, key) returns val.
func chiParam(handler http.HandlerFunc, key, val string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add(key, val)
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
		handler(w, r)
	})
}

func decodeBodyMap(t *testing.T, rr *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&m); err != nil {
		t.Fatalf("failed to decode response body: %v (body: %s)", err, rr.Body.String())
	}
	return m
}

var (
	testUserID    = uuid.New()
	testCompanyID = uuid.New()
)

// ── handleCreateHire ──────────────────────────────────────────────────────────

func TestHandleCreateHire_NoAuth_Returns401(t *testing.T) {
	s := newHiringTestServer(newMockStore(), newMockOrch())
	req := reqNoAuth("POST", "/hires", []byte(`{"role_title":"eng","runtime":"claude_code"}`))
	rr := httptest.NewRecorder()
	s.authMiddleware(http.HandlerFunc(s.handleCreateHire)).ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestHandleCreateHire_InvalidJSON_Returns400(t *testing.T) {
	s := newHiringTestServer(newMockStore(), newMockOrch())
	req := reqWithAuth("POST", "/hires", []byte(`{bad json`), testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	s.handleCreateHire(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleCreateHire_MissingRoleTitle_Returns400(t *testing.T) {
	s := newHiringTestServer(newMockStore(), newMockOrch())
	body := `{"runtime":"claude_code"}`
	req := reqWithAuth("POST", "/hires", []byte(body), testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	s.handleCreateHire(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
	m := decodeBodyMap(t, rr)
	if m["error"] != "role_title is required" {
		t.Errorf("expected 'role_title is required', got %v", m["error"])
	}
}

func TestHandleCreateHire_MissingRuntime_Returns400(t *testing.T) {
	s := newHiringTestServer(newMockStore(), newMockOrch())
	body := `{"role_title":"engineer"}`
	req := reqWithAuth("POST", "/hires", []byte(body), testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	s.handleCreateHire(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
	m := decodeBodyMap(t, rr)
	if m["error"] != "runtime is required" {
		t.Errorf("expected 'runtime is required', got %v", m["error"])
	}
}

func TestHandleCreateHire_ValidMinimalBody_Returns201(t *testing.T) {
	hireID := uuid.New()
	ms := newMockStore()
	ms.createPendingHireFn = func(_ context.Context, h *store.PendingHire) (*store.PendingHire, error) {
		h.ID = hireID
		h.Status = store.HirePending
		h.CreatedAt = time.Now()
		return h, nil
	}
	s := newHiringTestServer(ms, newMockOrch())
	body := `{"role_title":"engineer","runtime":"claude_code"}`
	req := reqWithAuth("POST", "/hires", []byte(body), testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	s.handleCreateHire(rr, req)
	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d (body: %s)", rr.Code, rr.Body.String())
	}
}

func TestHandleCreateHire_ValidBodyWithInitialTask_Returns201(t *testing.T) {
	hireID := uuid.New()
	ms := newMockStore()
	ms.createPendingHireFn = func(_ context.Context, h *store.PendingHire) (*store.PendingHire, error) {
		h.ID = hireID
		h.Status = store.HirePending
		return h, nil
	}
	s := newHiringTestServer(ms, newMockOrch())
	body := `{"role_title":"engineer","runtime":"claude_code","initial_task":"write tests"}`
	req := reqWithAuth("POST", "/hires", []byte(body), testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	s.handleCreateHire(rr, req)
	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d (body: %s)", rr.Code, rr.Body.String())
	}
}

func TestHandleCreateHire_BudgetZeroIsAllowed(t *testing.T) {
	ms := newMockStore()
	ms.createPendingHireFn = func(_ context.Context, h *store.PendingHire) (*store.PendingHire, error) {
		if h.BudgetAllocation != 0 {
			return nil, fmt.Errorf("expected 0 budget, got %d", h.BudgetAllocation)
		}
		h.ID = uuid.New()
		h.Status = store.HirePending
		return h, nil
	}
	s := newHiringTestServer(ms, newMockOrch())
	body := `{"role_title":"engineer","runtime":"claude_code","budget_allocation":0}`
	req := reqWithAuth("POST", "/hires", []byte(body), testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	s.handleCreateHire(rr, req)
	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d (body: %s)", rr.Code, rr.Body.String())
	}
}

func TestHandleCreateHire_DBError_Returns500(t *testing.T) {
	ms := newMockStore()
	ms.createPendingHireFn = func(_ context.Context, h *store.PendingHire) (*store.PendingHire, error) {
		return nil, errors.New("db connection lost")
	}
	s := newHiringTestServer(ms, newMockOrch())
	body := `{"role_title":"engineer","runtime":"claude_code"}`
	req := reqWithAuth("POST", "/hires", []byte(body), testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	s.handleCreateHire(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

func TestHandleCreateHire_AllFieldsPopulated(t *testing.T) {
	reqByID := uuid.New()
	reportingID := uuid.New()
	var captured *store.PendingHire
	ms := newMockStore()
	ms.createPendingHireFn = func(_ context.Context, h *store.PendingHire) (*store.PendingHire, error) {
		captured = h
		h.ID = uuid.New()
		h.Status = store.HirePending
		return h, nil
	}
	s := newHiringTestServer(ms, newMockOrch())
	task := "build microservice"
	body := fmt.Sprintf(`{
		"requested_by_agent_id": %q,
		"role_title": "senior engineer",
		"reporting_to_agent_id": %q,
		"system_prompt": "you are an expert",
		"runtime": "openclaw",
		"budget_allocation": 5000,
		"initial_task": %q
	}`, reqByID.String(), reportingID.String(), task)
	req := reqWithAuth("POST", "/hires", []byte(body), testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	s.handleCreateHire(rr, req)
	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d (body: %s)", rr.Code, rr.Body.String())
	}
	if captured == nil {
		t.Fatal("createPendingHire was not called")
	}
	if captured.RoleTitle != "senior engineer" {
		t.Errorf("RoleTitle: expected 'senior engineer', got %q", captured.RoleTitle)
	}
	if captured.Runtime != store.AgentRuntime("openclaw") {
		t.Errorf("Runtime: expected openclaw, got %q", captured.Runtime)
	}
	if captured.BudgetAllocation != 5000 {
		t.Errorf("BudgetAllocation: expected 5000, got %d", captured.BudgetAllocation)
	}
	if captured.SystemPrompt != "you are an expert" {
		t.Errorf("SystemPrompt: expected 'you are an expert', got %q", captured.SystemPrompt)
	}
	if captured.RequestedByAgentID != reqByID {
		t.Errorf("RequestedByAgentID: expected %s, got %s", reqByID, captured.RequestedByAgentID)
	}
	if captured.ReportingToAgentID != reportingID {
		t.Errorf("ReportingToAgentID: expected %s, got %s", reportingID, captured.ReportingToAgentID)
	}
	if captured.InitialTask == nil || *captured.InitialTask != task {
		t.Errorf("InitialTask: expected %q, got %v", task, captured.InitialTask)
	}
	if captured.CompanyID != testCompanyID {
		t.Errorf("CompanyID: expected %s, got %s", testCompanyID, captured.CompanyID)
	}
}

func TestHandleCreateHire_ResponseContainsID(t *testing.T) {
	hireID := uuid.New()
	ms := newMockStore()
	ms.createPendingHireFn = func(_ context.Context, h *store.PendingHire) (*store.PendingHire, error) {
		h.ID = hireID
		h.Status = store.HirePending
		return h, nil
	}
	s := newHiringTestServer(ms, newMockOrch())
	body := `{"role_title":"eng","runtime":"claude_code"}`
	req := reqWithAuth("POST", "/hires", []byte(body), testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	s.handleCreateHire(rr, req)
	m := decodeBodyMap(t, rr)
	if m["id"] != hireID.String() {
		t.Errorf("expected id %s in response, got %v", hireID, m["id"])
	}
}

func TestHandleCreateHire_RuntimeCopiedToHire(t *testing.T) {
	var capturedRuntime store.AgentRuntime
	ms := newMockStore()
	ms.createPendingHireFn = func(_ context.Context, h *store.PendingHire) (*store.PendingHire, error) {
		capturedRuntime = h.Runtime
		h.ID = uuid.New()
		h.Status = store.HirePending
		return h, nil
	}
	s := newHiringTestServer(ms, newMockOrch())
	body := `{"role_title":"eng","runtime":"openclaw"}`
	req := reqWithAuth("POST", "/hires", []byte(body), testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	s.handleCreateHire(rr, req)
	if capturedRuntime != store.AgentRuntime("openclaw") {
		t.Errorf("expected runtime openclaw, got %q", capturedRuntime)
	}
}

func TestHandleCreateHire_NilInitialTaskWhenOmitted(t *testing.T) {
	var capturedTask *string
	ms := newMockStore()
	ms.createPendingHireFn = func(_ context.Context, h *store.PendingHire) (*store.PendingHire, error) {
		capturedTask = h.InitialTask
		h.ID = uuid.New()
		h.Status = store.HirePending
		return h, nil
	}
	s := newHiringTestServer(ms, newMockOrch())
	body := `{"role_title":"eng","runtime":"claude_code"}`
	req := reqWithAuth("POST", "/hires", []byte(body), testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	s.handleCreateHire(rr, req)
	if capturedTask != nil {
		t.Errorf("expected nil InitialTask when omitted, got %v", *capturedTask)
	}
}

// ── handleListPendingHires ────────────────────────────────────────────────────

func TestHandleListPendingHires_NoAuth_Returns401(t *testing.T) {
	s := newHiringTestServer(newMockStore(), newMockOrch())
	req := reqNoAuth("GET", "/hires", nil)
	rr := httptest.NewRecorder()
	s.authMiddleware(http.HandlerFunc(s.handleListPendingHires)).ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestHandleListPendingHires_EmptyDB_Returns200EmptyArray(t *testing.T) {
	ms := newMockStore()
	ms.listPendingHiresFn = func(_ context.Context, _ uuid.UUID) ([]store.PendingHire, error) {
		return []store.PendingHire{}, nil
	}
	s := newHiringTestServer(ms, newMockOrch())
	req := reqWithAuth("GET", "/hires", nil, testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	s.handleListPendingHires(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var result []interface{}
	json.NewDecoder(rr.Body).Decode(&result)
	if len(result) != 0 {
		t.Errorf("expected empty array, got %d elements", len(result))
	}
}

func TestHandleListPendingHires_NilFromDB_Returns200EmptyArray(t *testing.T) {
	ms := newMockStore()
	ms.listPendingHiresFn = func(_ context.Context, _ uuid.UUID) ([]store.PendingHire, error) {
		return nil, nil
	}
	s := newHiringTestServer(ms, newMockOrch())
	req := reqWithAuth("GET", "/hires", nil, testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	s.handleListPendingHires(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	// Must not be "null"
	if body == "null\n" || body == "null" {
		t.Errorf("expected [] not null in response body, got %q", body)
	}
	var result []interface{}
	json.NewDecoder(bytes.NewBufferString(body)).Decode(&result)
	if result == nil {
		t.Errorf("expected non-nil decoded array")
	}
}

func TestHandleListPendingHires_WithHires_Returns200WithData(t *testing.T) {
	hireID := uuid.New()
	ms := newMockStore()
	ms.listPendingHiresFn = func(_ context.Context, _ uuid.UUID) ([]store.PendingHire, error) {
		return []store.PendingHire{
			{
				ID:        hireID,
				CompanyID: testCompanyID,
				RoleTitle: "engineer",
				Runtime:   store.RuntimeClaudeCode,
				Status:    store.HirePending,
			},
		}, nil
	}
	s := newHiringTestServer(ms, newMockOrch())
	req := reqWithAuth("GET", "/hires", nil, testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	s.handleListPendingHires(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var result []map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&result)
	if len(result) != 1 {
		t.Fatalf("expected 1 hire, got %d", len(result))
	}
	if result[0]["id"] != hireID.String() {
		t.Errorf("expected hire id %s, got %v", hireID, result[0]["id"])
	}
}

func TestHandleListPendingHires_DBError_Returns500(t *testing.T) {
	ms := newMockStore()
	ms.listPendingHiresFn = func(_ context.Context, _ uuid.UUID) ([]store.PendingHire, error) {
		return nil, errors.New("db error")
	}
	s := newHiringTestServer(ms, newMockOrch())
	req := reqWithAuth("GET", "/hires", nil, testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	s.handleListPendingHires(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

func TestHandleListPendingHires_MultipleHires_AllReturned(t *testing.T) {
	ms := newMockStore()
	ms.listPendingHiresFn = func(_ context.Context, _ uuid.UUID) ([]store.PendingHire, error) {
		return []store.PendingHire{
			{ID: uuid.New(), RoleTitle: "eng1", Runtime: store.RuntimeClaudeCode, Status: store.HirePending},
			{ID: uuid.New(), RoleTitle: "eng2", Runtime: store.RuntimeClaudeCode, Status: store.HirePending},
			{ID: uuid.New(), RoleTitle: "eng3", Runtime: store.RuntimeClaudeCode, Status: store.HirePending},
		}, nil
	}
	s := newHiringTestServer(ms, newMockOrch())
	req := reqWithAuth("GET", "/hires", nil, testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	s.handleListPendingHires(rr, req)
	var result []interface{}
	json.NewDecoder(rr.Body).Decode(&result)
	if len(result) != 3 {
		t.Errorf("expected 3 hires, got %d", len(result))
	}
}

// ── handleApproveHire ─────────────────────────────────────────────────────────

func TestHandleApproveHire_NoAuth_Returns401(t *testing.T) {
	s := newHiringTestServer(newMockStore(), newMockOrch())
	req := reqNoAuth("POST", "/hires/"+uuid.New().String()+"/approve", nil)
	rr := httptest.NewRecorder()
	s.authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chiParam(s.handleApproveHire, "hireID", uuid.New().String()).ServeHTTP(w, r)
	})).ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestHandleApproveHire_InvalidHireUUID_Returns400(t *testing.T) {
	s := newHiringTestServer(newMockStore(), newMockOrch())
	req := reqWithAuth("POST", "/hires/not-a-uuid/approve", nil, testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	chiParam(s.handleApproveHire, "hireID", "not-a-uuid").ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleApproveHire_HireNotFound_Returns404(t *testing.T) {
	mo := newMockOrch()
	mo.approveHireFn = func(_ context.Context, _ uuid.UUID) error {
		return errors.New("no rows in result set")
	}
	s := newHiringTestServer(newMockStore(), mo)
	hireID := uuid.New()
	req := reqWithAuth("POST", "/hires/"+hireID.String()+"/approve", nil, testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	chiParam(s.handleApproveHire, "hireID", hireID.String()).ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d (body: %s)", rr.Code, rr.Body.String())
	}
	m := decodeBodyMap(t, rr)
	if m["error"] != "hire not found" {
		t.Errorf("expected 'hire not found', got %v", m["error"])
	}
}

func TestHandleApproveHire_AlreadyApproved_Returns409(t *testing.T) {
	mo := newMockOrch()
	mo.approveHireFn = func(_ context.Context, _ uuid.UUID) error {
		return errors.New("hire is not pending")
	}
	s := newHiringTestServer(newMockStore(), mo)
	hireID := uuid.New()
	req := reqWithAuth("POST", "/hires/"+hireID.String()+"/approve", nil, testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	chiParam(s.handleApproveHire, "hireID", hireID.String()).ServeHTTP(rr, req)
	if rr.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d (body: %s)", rr.Code, rr.Body.String())
	}
}

func TestHandleApproveHire_OrchError_Returns500(t *testing.T) {
	mo := newMockOrch()
	mo.approveHireFn = func(_ context.Context, _ uuid.UUID) error {
		return errors.New("unexpected failure")
	}
	s := newHiringTestServer(newMockStore(), mo)
	hireID := uuid.New()
	req := reqWithAuth("POST", "/hires/"+hireID.String()+"/approve", nil, testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	chiParam(s.handleApproveHire, "hireID", hireID.String()).ServeHTTP(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

func TestHandleApproveHire_Success_Returns200(t *testing.T) {
	mo := newMockOrch()
	mo.approveHireFn = func(_ context.Context, _ uuid.UUID) error { return nil }
	s := newHiringTestServer(newMockStore(), mo)
	hireID := uuid.New()
	req := reqWithAuth("POST", "/hires/"+hireID.String()+"/approve", nil, testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	chiParam(s.handleApproveHire, "hireID", hireID.String()).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}
	m := decodeBodyMap(t, rr)
	if m["status"] != "approved" {
		t.Errorf("expected status=approved, got %v", m["status"])
	}
}

func TestHandleApproveHire_NoRowsVariant_Returns404(t *testing.T) {
	// "no rows" can also appear as "sql: no rows in result set"
	mo := newMockOrch()
	mo.approveHireFn = func(_ context.Context, _ uuid.UUID) error {
		return errors.New("sql: no rows in result set")
	}
	s := newHiringTestServer(newMockStore(), mo)
	hireID := uuid.New()
	req := reqWithAuth("POST", "/hires/"+hireID.String()+"/approve", nil, testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	chiParam(s.handleApproveHire, "hireID", hireID.String()).ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 for 'no rows', got %d", rr.Code)
	}
}

func TestHandleApproveHire_CorrectHireIDPassedToOrch(t *testing.T) {
	hireID := uuid.New()
	var passedID uuid.UUID
	mo := newMockOrch()
	mo.approveHireFn = func(_ context.Context, id uuid.UUID) error {
		passedID = id
		return nil
	}
	s := newHiringTestServer(newMockStore(), mo)
	req := reqWithAuth("POST", "/hires/"+hireID.String()+"/approve", nil, testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	chiParam(s.handleApproveHire, "hireID", hireID.String()).ServeHTTP(rr, req)
	if passedID != hireID {
		t.Errorf("expected hireID %s passed to orch, got %s", hireID, passedID)
	}
}

// ── handleRejectHire ──────────────────────────────────────────────────────────

func TestHandleRejectHire_NoAuth_Returns401(t *testing.T) {
	s := newHiringTestServer(newMockStore(), newMockOrch())
	req := reqNoAuth("POST", "/hires/"+uuid.New().String()+"/reject", nil)
	rr := httptest.NewRecorder()
	s.authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chiParam(s.handleRejectHire, "hireID", uuid.New().String()).ServeHTTP(w, r)
	})).ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestHandleRejectHire_InvalidHireUUID_Returns400(t *testing.T) {
	s := newHiringTestServer(newMockStore(), newMockOrch())
	req := reqWithAuth("POST", "/hires/bad/reject", nil, testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	chiParam(s.handleRejectHire, "hireID", "bad").ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleRejectHire_OrchError_Returns500(t *testing.T) {
	mo := newMockOrch()
	mo.rejectHireFn = func(_ context.Context, _ uuid.UUID, _ string) error {
		return errors.New("db error")
	}
	s := newHiringTestServer(newMockStore(), mo)
	hireID := uuid.New()
	req := reqWithAuth("POST", "/hires/"+hireID.String()+"/reject", nil, testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	chiParam(s.handleRejectHire, "hireID", hireID.String()).ServeHTTP(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

func TestHandleRejectHire_Success_Returns200(t *testing.T) {
	mo := newMockOrch()
	mo.rejectHireFn = func(_ context.Context, _ uuid.UUID, _ string) error { return nil }
	s := newHiringTestServer(newMockStore(), mo)
	hireID := uuid.New()
	req := reqWithAuth("POST", "/hires/"+hireID.String()+"/reject", nil, testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	chiParam(s.handleRejectHire, "hireID", hireID.String()).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}
	m := decodeBodyMap(t, rr)
	if m["status"] != "rejected" {
		t.Errorf("expected status=rejected, got %v", m["status"])
	}
}

func TestHandleRejectHire_WithReasonBody(t *testing.T) {
	var capturedReason string
	mo := newMockOrch()
	mo.rejectHireFn = func(_ context.Context, _ uuid.UUID, reason string) error {
		capturedReason = reason
		return nil
	}
	s := newHiringTestServer(newMockStore(), mo)
	hireID := uuid.New()
	body := `{"reason":"budget constraints"}`
	req := reqWithAuth("POST", "/hires/"+hireID.String()+"/reject", []byte(body), testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	chiParam(s.handleRejectHire, "hireID", hireID.String()).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if capturedReason != "budget constraints" {
		t.Errorf("expected reason 'budget constraints', got %q", capturedReason)
	}
}

func TestHandleRejectHire_EmptyReasonBody(t *testing.T) {
	var capturedReason string
	mo := newMockOrch()
	mo.rejectHireFn = func(_ context.Context, _ uuid.UUID, reason string) error {
		capturedReason = reason
		return nil
	}
	s := newHiringTestServer(newMockStore(), mo)
	hireID := uuid.New()
	body := `{"reason":""}`
	req := reqWithAuth("POST", "/hires/"+hireID.String()+"/reject", []byte(body), testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	chiParam(s.handleRejectHire, "hireID", hireID.String()).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if capturedReason != "" {
		t.Errorf("expected empty reason, got %q", capturedReason)
	}
}

func TestHandleRejectHire_InvalidJSONBodyStillWorks(t *testing.T) {
	// reason is optional; invalid JSON body is silently ignored (decodeJSON is nolint'd)
	mo := newMockOrch()
	mo.rejectHireFn = func(_ context.Context, _ uuid.UUID, _ string) error { return nil }
	s := newHiringTestServer(newMockStore(), mo)
	hireID := uuid.New()
	req := reqWithAuth("POST", "/hires/"+hireID.String()+"/reject", []byte(`{bad json`), testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	chiParam(s.handleRejectHire, "hireID", hireID.String()).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 even with bad JSON body, got %d (body: %s)", rr.Code, rr.Body.String())
	}
}

func TestHandleRejectHire_NoBodyStillWorks(t *testing.T) {
	mo := newMockOrch()
	mo.rejectHireFn = func(_ context.Context, _ uuid.UUID, _ string) error { return nil }
	s := newHiringTestServer(newMockStore(), mo)
	hireID := uuid.New()
	req := reqWithAuth("POST", "/hires/"+hireID.String()+"/reject", nil, testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	chiParam(s.handleRejectHire, "hireID", hireID.String()).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 with no body, got %d (body: %s)", rr.Code, rr.Body.String())
	}
}

func TestHandleRejectHire_CorrectHireIDPassedToOrch(t *testing.T) {
	hireID := uuid.New()
	var passedID uuid.UUID
	mo := newMockOrch()
	mo.rejectHireFn = func(_ context.Context, id uuid.UUID, _ string) error {
		passedID = id
		return nil
	}
	s := newHiringTestServer(newMockStore(), mo)
	req := reqWithAuth("POST", "/hires/"+hireID.String()+"/reject", nil, testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	chiParam(s.handleRejectHire, "hireID", hireID.String()).ServeHTTP(rr, req)
	if passedID != hireID {
		t.Errorf("expected hireID %s passed to orch, got %s", hireID, passedID)
	}
}

func TestHandleRejectHire_ContentTypeIsJSON(t *testing.T) {
	mo := newMockOrch()
	mo.rejectHireFn = func(_ context.Context, _ uuid.UUID, _ string) error { return nil }
	s := newHiringTestServer(newMockStore(), mo)
	hireID := uuid.New()
	req := reqWithAuth("POST", "/hires/"+hireID.String()+"/reject", nil, testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	chiParam(s.handleRejectHire, "hireID", hireID.String()).ServeHTTP(rr, req)
	ct := rr.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}
}
