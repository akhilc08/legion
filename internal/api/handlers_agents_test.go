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

// ── handleListAgents ──────────────────────────────────────────────────────────

func TestHandleListAgents_NoAuth(t *testing.T) {
	srv := newTestServer(newMockStore(), newMockOrch())
	rr := do(srv, "GET", companyPath(fixedCompanyID(), "/agents"), nil, nil)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestHandleListAgents_Empty(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	ms.listAgentsByCompanyFn = func(_ context.Context, _ uuid.UUID) ([]store.Agent, error) {
		return []store.Agent{}, nil
	}
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "GET", companyPath(fixedCompanyID(), "/agents"), nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var result []store.Agent
	decodeBodyInto(rr, &result) //nolint
	if len(result) != 0 {
		t.Fatalf("expected empty slice, got %d", len(result))
	}
}

func TestHandleListAgents_Populated(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	cid := fixedCompanyID()
	aid := fixedAgentID()
	ms.listAgentsByCompanyFn = func(_ context.Context, _ uuid.UUID) ([]store.Agent, error) {
		return []store.Agent{*sampleAgent(aid, cid)}, nil
	}
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "GET", companyPath(cid, "/agents"), nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var result []store.Agent
	decodeBodyInto(rr, &result) //nolint
	if len(result) != 1 || result[0].ID != aid {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestHandleListAgents_NilReturnedAsEmptySlice(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	ms.listAgentsByCompanyFn = func(_ context.Context, _ uuid.UUID) ([]store.Agent, error) {
		return nil, nil
	}
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "GET", companyPath(fixedCompanyID(), "/agents"), nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if body == "null\n" {
		t.Fatal("response must not be null")
	}
}

func TestHandleListAgents_DBError(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	ms.listAgentsByCompanyFn = func(_ context.Context, _ uuid.UUID) ([]store.Agent, error) {
		return nil, errors.New("db error")
	}
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "GET", companyPath(fixedCompanyID(), "/agents"), nil)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

// ── handleCreateAgent ─────────────────────────────────────────────────────────

func TestHandleCreateAgent_NoAuth(t *testing.T) {
	srv := newTestServer(newMockStore(), newMockOrch())
	rr := do(srv, "POST", companyPath(fixedCompanyID(), "/agents"), map[string]string{"role": "eng"}, nil)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestHandleCreateAgent_InvalidJSON(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	srv := newTestServer(ms, newMockOrch())

	req, _ := newRawRequest("POST", companyPath(fixedCompanyID(), "/agents"), []byte("{bad"), map[string]string{
		"Authorization": authHeader(fixedUserID()),
	})
	rr := httptest.NewRecorder()
	srv.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandleCreateAgent_DefaultRuntime(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	cid := fixedCompanyID()
	var captured *store.Agent

	ms.createAgentFn = func(_ context.Context, a *store.Agent) (*store.Agent, error) {
		captured = a
		return sampleAgent(fixedAgentID(), cid), nil
	}
	srv := newTestServer(ms, newMockOrch())

	doAuth(srv, fixedUserID(), "POST", companyPath(cid, "/agents"), map[string]interface{}{
		"role":  "engineer",
		"title": "Eng",
	})

	if captured == nil {
		t.Fatal("CreateAgent not called")
	}
	if captured.Runtime != store.RuntimeClaudeCode {
		t.Fatalf("expected default runtime claude_code, got %q", captured.Runtime)
	}
}

func TestHandleCreateAgent_DefaultBudget(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	cid := fixedCompanyID()
	var captured *store.Agent

	ms.createAgentFn = func(_ context.Context, a *store.Agent) (*store.Agent, error) {
		captured = a
		return sampleAgent(fixedAgentID(), cid), nil
	}
	srv := newTestServer(ms, newMockOrch())

	doAuth(srv, fixedUserID(), "POST", companyPath(cid, "/agents"), map[string]interface{}{
		"role": "eng",
	})

	if captured == nil {
		t.Fatal("CreateAgent not called")
	}
	if captured.MonthlyBudget != 100000 {
		t.Fatalf("expected default budget 100000, got %d", captured.MonthlyBudget)
	}
}

func TestHandleCreateAgent_CustomBudget(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	cid := fixedCompanyID()
	var captured *store.Agent

	ms.createAgentFn = func(_ context.Context, a *store.Agent) (*store.Agent, error) {
		captured = a
		return sampleAgent(fixedAgentID(), cid), nil
	}
	srv := newTestServer(ms, newMockOrch())

	doAuth(srv, fixedUserID(), "POST", companyPath(cid, "/agents"), map[string]interface{}{
		"role":           "eng",
		"monthly_budget": 5000,
	})

	if captured == nil {
		t.Fatal("CreateAgent not called")
	}
	if captured.MonthlyBudget != 5000 {
		t.Fatalf("expected budget 5000, got %d", captured.MonthlyBudget)
	}
}

func TestHandleCreateAgent_DBError(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	ms.createAgentFn = func(_ context.Context, _ *store.Agent) (*store.Agent, error) {
		return nil, errors.New("db error")
	}
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "POST", companyPath(fixedCompanyID(), "/agents"), map[string]interface{}{
		"role": "eng",
	})
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestHandleCreateAgent_Success(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	cid := fixedCompanyID()
	aid := fixedAgentID()

	ms.createAgentFn = func(_ context.Context, a *store.Agent) (*store.Agent, error) {
		return sampleAgent(aid, cid), nil
	}
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "POST", companyPath(cid, "/agents"), map[string]interface{}{
		"role":  "engineer",
		"title": "Engineer",
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var agent store.Agent
	decodeBodyInto(rr, &agent) //nolint
	if agent.ID != aid {
		t.Fatalf("unexpected agent id %s", agent.ID)
	}
}

// ── handleGetAgent ────────────────────────────────────────────────────────────

func TestHandleGetAgent_NoAuth(t *testing.T) {
	srv := newTestServer(newMockStore(), newMockOrch())
	rr := do(srv, "GET", companyPath(fixedCompanyID(), "/agents/"+fixedAgentID().String()), nil, nil)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestHandleGetAgent_InvalidUUID(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "GET", companyPath(fixedCompanyID(), "/agents/not-a-uuid"), nil)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandleGetAgent_NotFound(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	ms.getAgentFn = func(_ context.Context, _ uuid.UUID) (*store.Agent, error) {
		return nil, errors.New("not found")
	}
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "GET", companyPath(fixedCompanyID(), "/agents/"+fixedAgentID().String()), nil)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestHandleGetAgent_Found(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	cid := fixedCompanyID()
	aid := fixedAgentID()
	ms.getAgentFn = func(_ context.Context, _ uuid.UUID) (*store.Agent, error) {
		return sampleAgent(aid, cid), nil
	}
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "GET", companyPath(cid, "/agents/"+aid.String()), nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var agent store.Agent
	decodeBodyInto(rr, &agent) //nolint
	if agent.ID != aid {
		t.Fatalf("unexpected id %s", agent.ID)
	}
}

// ── handleDeleteAgent ─────────────────────────────────────────────────────────

func TestHandleDeleteAgent_NoAuth(t *testing.T) {
	srv := newTestServer(newMockStore(), newMockOrch())
	rr := do(srv, "DELETE", companyPath(fixedCompanyID(), "/agents/"+fixedAgentID().String()), nil, nil)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestHandleDeleteAgent_InvalidUUID(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "DELETE", companyPath(fixedCompanyID(), "/agents/not-a-uuid"), nil)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandleDeleteAgent_DBError(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	ms.deleteAgentFn = func(_ context.Context, _ uuid.UUID) error { return errors.New("db error") }
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "DELETE", companyPath(fixedCompanyID(), "/agents/"+fixedAgentID().String()), nil)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestHandleDeleteAgent_Success(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	ms.deleteAgentFn = func(_ context.Context, _ uuid.UUID) error { return nil }
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "DELETE", companyPath(fixedCompanyID(), "/agents/"+fixedAgentID().String()), nil)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}
}

// ── handleSpawnAgent ──────────────────────────────────────────────────────────

func TestHandleSpawnAgent_NoAuth(t *testing.T) {
	srv := newTestServer(newMockStore(), newMockOrch())
	rr := do(srv, "POST", companyPath(fixedCompanyID(), "/agents/"+fixedAgentID().String()+"/spawn"), nil, nil)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestHandleSpawnAgent_InvalidUUID(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "POST", companyPath(fixedCompanyID(), "/agents/bad-id/spawn"), nil)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandleSpawnAgent_NotFound(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	ms.getAgentFn = func(_ context.Context, _ uuid.UUID) (*store.Agent, error) {
		return nil, errors.New("not found")
	}
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "POST", companyPath(fixedCompanyID(), "/agents/"+fixedAgentID().String()+"/spawn"), nil)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestHandleSpawnAgent_OrchError(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	mo := newMockOrch()
	cid := fixedCompanyID()
	aid := fixedAgentID()
	ms.getAgentFn = func(_ context.Context, _ uuid.UUID) (*store.Agent, error) {
		return sampleAgent(aid, cid), nil
	}
	mo.spawnAgentFn = func(_ context.Context, _ *store.Agent) error { return errors.New("spawn error") }
	srv := newTestServer(ms, mo)

	rr := doAuth(srv, fixedUserID(), "POST", companyPath(cid, "/agents/"+aid.String()+"/spawn"), nil)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestHandleSpawnAgent_Success(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	mo := newMockOrch()
	cid := fixedCompanyID()
	aid := fixedAgentID()
	ms.getAgentFn = func(_ context.Context, _ uuid.UUID) (*store.Agent, error) {
		return sampleAgent(aid, cid), nil
	}
	mo.spawnAgentFn = func(_ context.Context, _ *store.Agent) error { return nil }
	srv := newTestServer(ms, mo)

	rr := doAuth(srv, fixedUserID(), "POST", companyPath(cid, "/agents/"+aid.String()+"/spawn"), nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

// ── handleKillAgent ───────────────────────────────────────────────────────────

func TestHandleKillAgent_InvalidUUID(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "POST", companyPath(fixedCompanyID(), "/agents/bad-id/kill"), nil)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandleKillAgent_OrchError(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	mo := newMockOrch()
	mo.killAgentFn = func(_ context.Context, _ uuid.UUID) error { return errors.New("kill error") }
	srv := newTestServer(ms, mo)

	rr := doAuth(srv, fixedUserID(), "POST", companyPath(fixedCompanyID(), "/agents/"+fixedAgentID().String()+"/kill"), nil)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestHandleKillAgent_Success(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	mo := newMockOrch()
	mo.killAgentFn = func(_ context.Context, _ uuid.UUID) error { return nil }
	srv := newTestServer(ms, mo)

	rr := doAuth(srv, fixedUserID(), "POST", companyPath(fixedCompanyID(), "/agents/"+fixedAgentID().String()+"/kill"), nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

// ── handlePauseAgent ──────────────────────────────────────────────────────────

func TestHandlePauseAgent_InvalidUUID(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "POST", companyPath(fixedCompanyID(), "/agents/bad-id/pause"), nil)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandlePauseAgent_OrchError(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	mo := newMockOrch()
	mo.pauseAgentFn = func(_ context.Context, _ uuid.UUID) error { return errors.New("pause error") }
	srv := newTestServer(ms, mo)

	rr := doAuth(srv, fixedUserID(), "POST", companyPath(fixedCompanyID(), "/agents/"+fixedAgentID().String()+"/pause"), nil)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestHandlePauseAgent_Success(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	mo := newMockOrch()
	mo.pauseAgentFn = func(_ context.Context, _ uuid.UUID) error { return nil }
	srv := newTestServer(ms, mo)

	rr := doAuth(srv, fixedUserID(), "POST", companyPath(fixedCompanyID(), "/agents/"+fixedAgentID().String()+"/pause"), nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

// ── handleResumeAgent ─────────────────────────────────────────────────────────

func TestHandleResumeAgent_InvalidUUID(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "POST", companyPath(fixedCompanyID(), "/agents/bad-id/resume"), nil)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandleResumeAgent_OrchError(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	mo := newMockOrch()
	mo.resumeAgentFn = func(_ context.Context, _ uuid.UUID) error { return errors.New("resume error") }
	srv := newTestServer(ms, mo)

	rr := doAuth(srv, fixedUserID(), "POST", companyPath(fixedCompanyID(), "/agents/"+fixedAgentID().String()+"/resume"), nil)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestHandleResumeAgent_Success(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	mo := newMockOrch()
	mo.resumeAgentFn = func(_ context.Context, _ uuid.UUID) error { return nil }
	srv := newTestServer(ms, mo)

	rr := doAuth(srv, fixedUserID(), "POST", companyPath(fixedCompanyID(), "/agents/"+fixedAgentID().String()+"/resume"), nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

// ── handleReassignAgent ───────────────────────────────────────────────────────

func TestHandleReassignAgent_InvalidUUID(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "POST", companyPath(fixedCompanyID(), "/agents/bad-id/reassign"), map[string]string{
		"new_manager_id": uuid.New().String(),
	})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandleReassignAgent_InvalidJSON(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	srv := newTestServer(ms, newMockOrch())

	req, _ := newRawRequest("POST", companyPath(fixedCompanyID(), "/agents/"+fixedAgentID().String()+"/reassign"), []byte("{bad"), map[string]string{
		"Authorization": authHeader(fixedUserID()),
	})
	rr := httptest.NewRecorder()
	srv.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandleReassignAgent_OrchError(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	mo := newMockOrch()
	mo.reassignAgentFn = func(_ context.Context, _, _ uuid.UUID) error { return errors.New("reassign error") }
	srv := newTestServer(ms, mo)

	rr := doAuth(srv, fixedUserID(), "POST", companyPath(fixedCompanyID(), "/agents/"+fixedAgentID().String()+"/reassign"), map[string]string{
		"new_manager_id": uuid.New().String(),
	})
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestHandleReassignAgent_Success(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	mo := newMockOrch()
	mo.reassignAgentFn = func(_ context.Context, _, _ uuid.UUID) error { return nil }
	srv := newTestServer(ms, mo)

	rr := doAuth(srv, fixedUserID(), "POST", companyPath(fixedCompanyID(), "/agents/"+fixedAgentID().String()+"/reassign"), map[string]string{
		"new_manager_id": uuid.New().String(),
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

// ── handleAgentChat ───────────────────────────────────────────────────────────

func TestHandleAgentChat_InvalidUUID(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "POST", companyPath(fixedCompanyID(), "/agents/bad-id/chat"), map[string]string{"message": "hello"})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandleAgentChat_InvalidJSON(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	srv := newTestServer(ms, newMockOrch())

	req, _ := newRawRequest("POST", companyPath(fixedCompanyID(), "/agents/"+fixedAgentID().String()+"/chat"), []byte("{bad"), map[string]string{
		"Authorization": authHeader(fixedUserID()),
	})
	rr := httptest.NewRecorder()
	srv.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandleAgentChat_OrchError(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	mo := newMockOrch()
	mo.sendChatMessageFn = func(_ context.Context, _ uuid.UUID, _ string) (string, error) {
		return "", errors.New("chat error")
	}
	srv := newTestServer(ms, mo)

	rr := doAuth(srv, fixedUserID(), "POST", companyPath(fixedCompanyID(), "/agents/"+fixedAgentID().String()+"/chat"), map[string]string{"message": "hi"})
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestHandleAgentChat_Success(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	mo := newMockOrch()
	mo.sendChatMessageFn = func(_ context.Context, _ uuid.UUID, _ string) (string, error) {
		return "hello back", nil
	}
	ms.getOrCreateChatSessionFn = func(_ context.Context, _ uuid.UUID) (*store.ChatSession, error) {
		return &store.ChatSession{ID: uuid.New(), AgentID: fixedAgentID(), StartedAt: time.Now()}, nil
	}
	ms.appendChatMessageFn = func(_ context.Context, _ uuid.UUID, _, _ string) error { return nil }
	srv := newTestServer(ms, mo)

	rr := doAuth(srv, fixedUserID(), "POST", companyPath(fixedCompanyID(), "/agents/"+fixedAgentID().String()+"/chat"), map[string]string{"message": "hi"})
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]string
	decodeBodyInto(rr, &resp) //nolint
	if resp["reply"] != "hello back" {
		t.Fatalf("unexpected reply %q", resp["reply"])
	}
}

// ── handleChatHistory ─────────────────────────────────────────────────────────

func TestHandleChatHistory_InvalidUUID(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "GET", companyPath(fixedCompanyID(), "/agents/bad-id/chat/history"), nil)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandleChatHistory_DBError(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	ms.getChatHistoryFn = func(_ context.Context, _ uuid.UUID) ([]store.ChatMessage, error) {
		return nil, errors.New("db error")
	}
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "GET", companyPath(fixedCompanyID(), "/agents/"+fixedAgentID().String()+"/chat/history"), nil)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestHandleChatHistory_Success(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	ms.getChatHistoryFn = func(_ context.Context, _ uuid.UUID) ([]store.ChatMessage, error) {
		return []store.ChatMessage{
			{Role: "user", Content: "hello", Timestamp: time.Now()},
			{Role: "assistant", Content: "hi", Timestamp: time.Now()},
		}, nil
	}
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "GET", companyPath(fixedCompanyID(), "/agents/"+fixedAgentID().String()+"/chat/history"), nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var msgs []store.ChatMessage
	decodeBodyInto(rr, &msgs) //nolint
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
}

func TestHandleChatHistory_EmptyHistory(t *testing.T) {
	ms := newMockStore()
	setupCompanyAccess(ms)
	ms.getChatHistoryFn = func(_ context.Context, _ uuid.UUID) ([]store.ChatMessage, error) {
		return []store.ChatMessage{}, nil
	}
	srv := newTestServer(ms, newMockOrch())

	rr := doAuth(srv, fixedUserID(), "GET", companyPath(fixedCompanyID(), "/agents/"+fixedAgentID().String()+"/chat/history"), nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}
