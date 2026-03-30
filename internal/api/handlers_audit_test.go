package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"conductor/internal/orchestrator"
	"conductor/internal/store"
	"github.com/google/uuid"
)

// ── handleListAuditLog ────────────────────────────────────────────────────────

func TestHandleListAuditLog_NoAuth_Returns401(t *testing.T) {
	s := newHiringTestServer(newMockStore(), newMockOrch())
	req := reqNoAuth("GET", "/audit", nil)
	rr := httptest.NewRecorder()
	s.authMiddleware(http.HandlerFunc(s.handleListAuditLog)).ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestHandleListAuditLog_NoLimitParam_DefaultsTo0(t *testing.T) {
	// When limit param is absent, strconv.Atoi returns 0 — the DB decides what to do.
	var capturedLimit int
	ms := newMockStore()
	ms.listAuditLogFn = func(_ context.Context, _ uuid.UUID, limit int) ([]store.AuditLog, error) {
		capturedLimit = limit
		return []store.AuditLog{}, nil
	}
	s := newHiringTestServer(ms, newMockOrch())
	req := reqWithAuth("GET", "/audit", nil, testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	s.handleListAuditLog(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if capturedLimit != 0 {
		t.Errorf("expected limit=0 (default) when param absent, got %d", capturedLimit)
	}
}

func TestHandleListAuditLog_Limit50_Passes50ToDB(t *testing.T) {
	var capturedLimit int
	ms := newMockStore()
	ms.listAuditLogFn = func(_ context.Context, _ uuid.UUID, limit int) ([]store.AuditLog, error) {
		capturedLimit = limit
		return []store.AuditLog{}, nil
	}
	s := newHiringTestServer(ms, newMockOrch())
	req := reqWithAuth("GET", "/audit?limit=50", nil, testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	s.handleListAuditLog(rr, req)
	if capturedLimit != 50 {
		t.Errorf("expected limit=50, got %d", capturedLimit)
	}
}

func TestHandleListAuditLog_Limit0_PassesToDB(t *testing.T) {
	var capturedLimit int
	ms := newMockStore()
	ms.listAuditLogFn = func(_ context.Context, _ uuid.UUID, limit int) ([]store.AuditLog, error) {
		capturedLimit = limit
		return []store.AuditLog{}, nil
	}
	s := newHiringTestServer(ms, newMockOrch())
	req := reqWithAuth("GET", "/audit?limit=0", nil, testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	s.handleListAuditLog(rr, req)
	// limit=0 parses to 0
	if capturedLimit != 0 {
		t.Errorf("expected limit=0 passed through, got %d", capturedLimit)
	}
}

func TestHandleListAuditLog_LimitNegative_PassesToDB(t *testing.T) {
	var capturedLimit int
	ms := newMockStore()
	ms.listAuditLogFn = func(_ context.Context, _ uuid.UUID, limit int) ([]store.AuditLog, error) {
		capturedLimit = limit
		return []store.AuditLog{}, nil
	}
	s := newHiringTestServer(ms, newMockOrch())
	req := reqWithAuth("GET", "/audit?limit=-1", nil, testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	s.handleListAuditLog(rr, req)
	// strconv.Atoi("-1") = -1, so -1 is passed to DB
	if capturedLimit != -1 {
		t.Errorf("expected limit=-1 passed through, got %d", capturedLimit)
	}
}

func TestHandleListAuditLog_LimitInvalidString_Passes0ToDB(t *testing.T) {
	var capturedLimit int
	ms := newMockStore()
	ms.listAuditLogFn = func(_ context.Context, _ uuid.UUID, limit int) ([]store.AuditLog, error) {
		capturedLimit = limit
		return []store.AuditLog{}, nil
	}
	s := newHiringTestServer(ms, newMockOrch())
	req := reqWithAuth("GET", "/audit?limit=abc", nil, testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	s.handleListAuditLog(rr, req)
	// strconv.Atoi("abc") fails, returns 0
	if capturedLimit != 0 {
		t.Errorf("expected limit=0 when param is non-numeric, got %d", capturedLimit)
	}
}

func TestHandleListAuditLog_NilFromDB_Returns200EmptyArray(t *testing.T) {
	ms := newMockStore()
	ms.listAuditLogFn = func(_ context.Context, _ uuid.UUID, _ int) ([]store.AuditLog, error) {
		return nil, nil
	}
	s := newHiringTestServer(ms, newMockOrch())
	req := reqWithAuth("GET", "/audit", nil, testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	s.handleListAuditLog(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if body == "null\n" || body == "null" {
		t.Errorf("expected [] not null, got %q", body)
	}
}

func TestHandleListAuditLog_WithLogs_Returns200WithData(t *testing.T) {
	logID := uuid.New()
	ms := newMockStore()
	ms.listAuditLogFn = func(_ context.Context, _ uuid.UUID, _ int) ([]store.AuditLog, error) {
		return []store.AuditLog{
			{
				ID:        logID,
				CompanyID: testCompanyID,
				EventType: "agent.spawned",
				Payload:   map[string]interface{}{"agent": "CEO"},
				CreatedAt: time.Now(),
			},
		}, nil
	}
	s := newHiringTestServer(ms, newMockOrch())
	req := reqWithAuth("GET", "/audit", nil, testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	s.handleListAuditLog(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var result []map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&result)
	if len(result) != 1 {
		t.Fatalf("expected 1 log, got %d", len(result))
	}
	if result[0]["id"] != logID.String() {
		t.Errorf("expected log id %s, got %v", logID, result[0]["id"])
	}
}

func TestHandleListAuditLog_DBError_Returns500(t *testing.T) {
	ms := newMockStore()
	ms.listAuditLogFn = func(_ context.Context, _ uuid.UUID, _ int) ([]store.AuditLog, error) {
		return nil, errors.New("db error")
	}
	s := newHiringTestServer(ms, newMockOrch())
	req := reqWithAuth("GET", "/audit", nil, testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	s.handleListAuditLog(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

func TestHandleListAuditLog_Limit100_Passes100ToDB(t *testing.T) {
	var capturedLimit int
	ms := newMockStore()
	ms.listAuditLogFn = func(_ context.Context, _ uuid.UUID, limit int) ([]store.AuditLog, error) {
		capturedLimit = limit
		return []store.AuditLog{}, nil
	}
	s := newHiringTestServer(ms, newMockOrch())
	req := reqWithAuth("GET", "/audit?limit=100", nil, testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	s.handleListAuditLog(rr, req)
	if capturedLimit != 100 {
		t.Errorf("expected limit=100, got %d", capturedLimit)
	}
}

func TestHandleListAuditLog_CompanyIDPassedToDB(t *testing.T) {
	var capturedCompanyID uuid.UUID
	ms := newMockStore()
	ms.listAuditLogFn = func(_ context.Context, companyID uuid.UUID, _ int) ([]store.AuditLog, error) {
		capturedCompanyID = companyID
		return []store.AuditLog{}, nil
	}
	s := newHiringTestServer(ms, newMockOrch())
	req := reqWithAuth("GET", "/audit", nil, testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	s.handleListAuditLog(rr, req)
	if capturedCompanyID != testCompanyID {
		t.Errorf("expected companyID %s, got %s", testCompanyID, capturedCompanyID)
	}
}

func TestHandleListAuditLog_ContentTypeIsJSON(t *testing.T) {
	ms := newMockStore()
	ms.listAuditLogFn = func(_ context.Context, _ uuid.UUID, _ int) ([]store.AuditLog, error) {
		return []store.AuditLog{}, nil
	}
	s := newHiringTestServer(ms, newMockOrch())
	req := reqWithAuth("GET", "/audit", nil, testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	s.handleListAuditLog(rr, req)
	ct := rr.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}
}

// ── handleListNotifications ───────────────────────────────────────────────────

func TestHandleListNotifications_NoAuth_Returns401(t *testing.T) {
	s := newHiringTestServer(newMockStore(), newMockOrch())
	req := reqNoAuth("GET", "/notifications", nil)
	rr := httptest.NewRecorder()
	s.authMiddleware(http.HandlerFunc(s.handleListNotifications)).ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestHandleListNotifications_NilFromDB_Returns200EmptyArray(t *testing.T) {
	ms := newMockStore()
	ms.listActiveNotificationsFn = func(_ context.Context, _ uuid.UUID) ([]store.Notification, error) {
		return nil, nil
	}
	s := newHiringTestServer(ms, newMockOrch())
	req := reqWithAuth("GET", "/notifications", nil, testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	s.handleListNotifications(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if body == "null\n" || body == "null" {
		t.Errorf("expected [] not null, got %q", body)
	}
}

func TestHandleListNotifications_WithNotifications_Returns200(t *testing.T) {
	notifID := uuid.New()
	ms := newMockStore()
	ms.listActiveNotificationsFn = func(_ context.Context, _ uuid.UUID) ([]store.Notification, error) {
		return []store.Notification{
			{
				ID:        notifID,
				CompanyID: testCompanyID,
				Type:      "hire.pending",
				CreatedAt: time.Now(),
			},
		}, nil
	}
	s := newHiringTestServer(ms, newMockOrch())
	req := reqWithAuth("GET", "/notifications", nil, testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	s.handleListNotifications(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var result []map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&result)
	if len(result) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(result))
	}
	if result[0]["id"] != notifID.String() {
		t.Errorf("expected notif id %s, got %v", notifID, result[0]["id"])
	}
}

func TestHandleListNotifications_DBError_Returns500(t *testing.T) {
	ms := newMockStore()
	ms.listActiveNotificationsFn = func(_ context.Context, _ uuid.UUID) ([]store.Notification, error) {
		return nil, errors.New("db error")
	}
	s := newHiringTestServer(ms, newMockOrch())
	req := reqWithAuth("GET", "/notifications", nil, testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	s.handleListNotifications(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

func TestHandleListNotifications_EmptyList_Returns200EmptyArray(t *testing.T) {
	ms := newMockStore()
	ms.listActiveNotificationsFn = func(_ context.Context, _ uuid.UUID) ([]store.Notification, error) {
		return []store.Notification{}, nil
	}
	s := newHiringTestServer(ms, newMockOrch())
	req := reqWithAuth("GET", "/notifications", nil, testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	s.handleListNotifications(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var result []interface{}
	json.NewDecoder(rr.Body).Decode(&result)
	if len(result) != 0 {
		t.Errorf("expected empty array, got %d", len(result))
	}
}

func TestHandleListNotifications_MultipleNotifications(t *testing.T) {
	ms := newMockStore()
	ms.listActiveNotificationsFn = func(_ context.Context, _ uuid.UUID) ([]store.Notification, error) {
		return []store.Notification{
			{ID: uuid.New(), Type: "type1", CompanyID: testCompanyID},
			{ID: uuid.New(), Type: "type2", CompanyID: testCompanyID},
		}, nil
	}
	s := newHiringTestServer(ms, newMockOrch())
	req := reqWithAuth("GET", "/notifications", nil, testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	s.handleListNotifications(rr, req)
	var result []interface{}
	json.NewDecoder(rr.Body).Decode(&result)
	if len(result) != 2 {
		t.Errorf("expected 2 notifications, got %d", len(result))
	}
}

// ── handleDismissNotification ─────────────────────────────────────────────────

func TestHandleDismissNotification_NoAuth_Returns401(t *testing.T) {
	s := newHiringTestServer(newMockStore(), newMockOrch())
	notifID := uuid.New()
	req := reqNoAuth("POST", "/notifications/"+notifID.String()+"/dismiss", nil)
	rr := httptest.NewRecorder()
	s.authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chiParam(s.handleDismissNotification, "notifID", notifID.String()).ServeHTTP(w, r)
	})).ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestHandleDismissNotification_InvalidUUID_Returns400(t *testing.T) {
	s := newHiringTestServer(newMockStore(), newMockOrch())
	req := reqWithAuth("POST", "/notifications/bad/dismiss", nil, testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	chiParam(s.handleDismissNotification, "notifID", "bad").ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleDismissNotification_DBError_Returns500(t *testing.T) {
	ms := newMockStore()
	ms.dismissNotificationFn = func(_ context.Context, _ uuid.UUID) error {
		return errors.New("db error")
	}
	s := newHiringTestServer(ms, newMockOrch())
	notifID := uuid.New()
	req := reqWithAuth("POST", "/notifications/"+notifID.String()+"/dismiss", nil, testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	chiParam(s.handleDismissNotification, "notifID", notifID.String()).ServeHTTP(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

func TestHandleDismissNotification_Success_Returns200OK(t *testing.T) {
	ms := newMockStore()
	ms.dismissNotificationFn = func(_ context.Context, _ uuid.UUID) error { return nil }
	s := newHiringTestServer(ms, newMockOrch())
	notifID := uuid.New()
	req := reqWithAuth("POST", "/notifications/"+notifID.String()+"/dismiss", nil, testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	chiParam(s.handleDismissNotification, "notifID", notifID.String()).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}
	m := decodeBody(t, rr)
	if m["status"] != "ok" {
		t.Errorf("expected status=ok, got %v", m["status"])
	}
}

func TestHandleDismissNotification_CorrectIDPassedToDB(t *testing.T) {
	notifID := uuid.New()
	var capturedID uuid.UUID
	ms := newMockStore()
	ms.dismissNotificationFn = func(_ context.Context, id uuid.UUID) error {
		capturedID = id
		return nil
	}
	s := newHiringTestServer(ms, newMockOrch())
	req := reqWithAuth("POST", "/notifications/"+notifID.String()+"/dismiss", nil, testUserID, testCompanyID)
	rr := httptest.NewRecorder()
	chiParam(s.handleDismissNotification, "notifID", notifID.String()).ServeHTTP(rr, req)
	if capturedID != notifID {
		t.Errorf("expected notifID %s, got %s", notifID, capturedID)
	}
}

// ── handleGetRuntimes ─────────────────────────────────────────────────────────

func TestHandleGetRuntimes_ClaudeCodeTrue_OpenClawFalse(t *testing.T) {
	mo := newMockOrch()
	mo.availableRuntimesFn = func() orchestrator.Runtimes {
		return orchestrator.Runtimes{ClaudeCode: true, OpenClaw: false}
	}
	s := newHiringTestServer(newMockStore(), mo)
	req := httptest.NewRequest("GET", "/api/runtimes", nil)
	rr := httptest.NewRecorder()
	s.handleGetRuntimes(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var result map[string]bool
	json.NewDecoder(rr.Body).Decode(&result)
	if !result["claude_code"] {
		t.Errorf("expected claude_code=true")
	}
	if result["openclaw"] {
		t.Errorf("expected openclaw=false")
	}
}

func TestHandleGetRuntimes_BothTrue(t *testing.T) {
	mo := newMockOrch()
	mo.availableRuntimesFn = func() orchestrator.Runtimes {
		return orchestrator.Runtimes{ClaudeCode: true, OpenClaw: true}
	}
	s := newHiringTestServer(newMockStore(), mo)
	req := httptest.NewRequest("GET", "/api/runtimes", nil)
	rr := httptest.NewRecorder()
	s.handleGetRuntimes(rr, req)
	var result map[string]bool
	json.NewDecoder(rr.Body).Decode(&result)
	if !result["claude_code"] {
		t.Errorf("expected claude_code=true")
	}
	if !result["openclaw"] {
		t.Errorf("expected openclaw=true")
	}
}

func TestHandleGetRuntimes_BothFalse(t *testing.T) {
	mo := newMockOrch()
	mo.availableRuntimesFn = func() orchestrator.Runtimes {
		return orchestrator.Runtimes{ClaudeCode: false, OpenClaw: false}
	}
	s := newHiringTestServer(newMockStore(), mo)
	req := httptest.NewRequest("GET", "/api/runtimes", nil)
	rr := httptest.NewRecorder()
	s.handleGetRuntimes(rr, req)
	var result map[string]bool
	json.NewDecoder(rr.Body).Decode(&result)
	if result["claude_code"] {
		t.Errorf("expected claude_code=false")
	}
	if result["openclaw"] {
		t.Errorf("expected openclaw=false")
	}
}

func TestHandleGetRuntimes_NoAuthRequired(t *testing.T) {
	// handleGetRuntimes should work without auth
	mo := newMockOrch()
	mo.availableRuntimesFn = func() orchestrator.Runtimes {
		return orchestrator.Runtimes{ClaudeCode: true, OpenClaw: false}
	}
	s := newHiringTestServer(newMockStore(), mo)
	// No auth context set at all
	req := httptest.NewRequest("GET", "/api/runtimes", nil)
	rr := httptest.NewRecorder()
	s.handleGetRuntimes(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 without auth, got %d", rr.Code)
	}
}

func TestHandleGetRuntimes_ResponseHasCorrectKeys(t *testing.T) {
	mo := newMockOrch()
	mo.availableRuntimesFn = func() orchestrator.Runtimes {
		return orchestrator.Runtimes{ClaudeCode: true, OpenClaw: true}
	}
	s := newHiringTestServer(newMockStore(), mo)
	req := httptest.NewRequest("GET", "/api/runtimes", nil)
	rr := httptest.NewRecorder()
	s.handleGetRuntimes(rr, req)
	var result map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&result)
	if _, ok := result["claude_code"]; !ok {
		t.Errorf("expected 'claude_code' key in response")
	}
	if _, ok := result["openclaw"]; !ok {
		t.Errorf("expected 'openclaw' key in response")
	}
}

// ── handleSPA ─────────────────────────────────────────────────────────────────

func TestHandleSPA_AttemptsToServeIndexHTML(t *testing.T) {
	// staticDir doesn't exist; http.ServeFile will return a 404.
	s := &Server{
		jwtSecret: "test-secret",
		staticDir: "/nonexistent-static-dir",
	}
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	s.handleSPA(rr, req)
	// http.ServeFile with a nonexistent path returns 404
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent static dir, got %d", rr.Code)
	}
}

func TestHandleSPA_WithEmptyStaticDir(t *testing.T) {
	s := &Server{
		jwtSecret: "test-secret",
		staticDir: "",
	}
	req := httptest.NewRequest("GET", "/dashboard", nil)
	rr := httptest.NewRecorder()
	s.handleSPA(rr, req)
	// ServeFile with empty path "/index.html" will fail
	if rr.Code == http.StatusOK {
		t.Errorf("expected non-200 for empty staticDir")
	}
}
