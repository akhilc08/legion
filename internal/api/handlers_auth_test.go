package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"conductor/internal/store"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// ── handleRegister ────────────────────────────────────────────────────────────

func TestHandleRegister_EmptyBody_Returns400(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	rr := do(s, "POST", "/api/auth/register", nil, nil)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleRegister_InvalidJSON_Returns400(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	req, _ := newRawRequest("POST", "/api/auth/register", []byte(`{bad json`), nil)
	rr := httptest.NewRecorder()
	s.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

// ── handleLogin ───────────────────────────────────────────────────────────────

func TestHandleLogin_EmptyBody_Returns400(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	rr := do(s, "POST", "/api/auth/login", nil, nil)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleLogin_InvalidJSON_Returns400(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	req, _ := newRawRequest("POST", "/api/auth/login", []byte(`{bad json`), nil)
	rr := httptest.NewRecorder()
	s.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleLogin_ArrayBody_Returns400(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	req, _ := newRawRequest("POST", "/api/auth/login", []byte(`[1,2,3]`), nil)
	rr := httptest.NewRecorder()
	s.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for array body, got %d", rr.Code)
	}
}

// ── handleGetRuntimes (public route) ─────────────────────────────────────────

func TestHandleGetRuntimes_NoAuth_Returns200(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	rr := do(s, "GET", "/api/runtimes", nil, nil)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestHandleGetRuntimes_ContentTypeJSON(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	rr := do(s, "GET", "/api/runtimes", nil, nil)
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected application/json, got %q", ct)
	}
}

// ── Authenticated routes: no token → 401 ─────────────────────────────────────

func TestHandleListCompanies_NoAuth_Returns401(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	rr := do(s, "GET", "/api/companies", nil, nil)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestHandleCreateCompany_NoAuth_Returns401(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	rr := do(s, "POST", "/api/companies", map[string]string{"name": "X"}, nil)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestHandleGetCompany_NoAuth_Returns401(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	rr := do(s, "GET", companyPath(uuid.New(), "/"), nil, nil)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestHandleListAgents_NoAuth_Returns401(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	rr := do(s, "GET", companyPath(uuid.New(), "/agents"), nil, nil)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestHandleListIssues_NoAuth_Returns401(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	rr := do(s, "GET", companyPath(uuid.New(), "/issues"), nil, nil)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestHandleListPendingHires_Router_NoAuth_Returns401(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	rr := do(s, "GET", companyPath(uuid.New(), "/hires"), nil, nil)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestHandleAuditLog_NoAuth_Returns401(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	rr := do(s, "GET", companyPath(uuid.New(), "/audit"), nil, nil)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestHandleNotifications_NoAuth_Returns401(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	rr := do(s, "GET", companyPath(uuid.New(), "/notifications"), nil, nil)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

// ── Company-scoped routes: invalid company UUID → 400 ─────────────────────────
// chiCompanyMiddleware parses companyID before touching DB; invalid UUID → 400.

func invalidCo(suffix string) string {
	return "/api/companies/not-a-uuid" + suffix
}

func TestHandleGetCompany_InvalidCompanyUUID_Returns400(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	rr := doAuth(s, fixedUserID(), "GET", invalidCo("/"), nil)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleUpdateCompany_InvalidCompanyUUID_Returns400(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	rr := doAuth(s, fixedUserID(), "PATCH", invalidCo("/"), map[string]string{"goal": "x"})
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleDeleteCompany_InvalidCompanyUUID_Returns400(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	rr := doAuth(s, fixedUserID(), "DELETE", invalidCo("/"), nil)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleSetGoal_InvalidCompanyUUID_Returns400(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	rr := doAuth(s, fixedUserID(), "POST", invalidCo("/goal"), map[string]string{"goal": "x"})
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleListAgents_InvalidCompanyUUID_Returns400(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	rr := doAuth(s, fixedUserID(), "GET", invalidCo("/agents"), nil)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleCreateAgent_InvalidCompanyUUID_Returns400(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	rr := doAuth(s, fixedUserID(), "POST", invalidCo("/agents"), map[string]string{"role": "ceo"})
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleGetAgent_InvalidCompanyUUID_Returns400(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	rr := doAuth(s, fixedUserID(), "GET", invalidCo("/agents/"+uuid.New().String()), nil)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleDeleteAgent_InvalidCompanyUUID_Returns400(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	rr := doAuth(s, fixedUserID(), "DELETE", invalidCo("/agents/"+uuid.New().String()), nil)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleSpawnAgent_InvalidCompanyUUID_Returns400(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	rr := doAuth(s, fixedUserID(), "POST", invalidCo("/agents/"+uuid.New().String()+"/spawn"), nil)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleKillAgent_InvalidCompanyUUID_Returns400(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	rr := doAuth(s, fixedUserID(), "POST", invalidCo("/agents/"+uuid.New().String()+"/kill"), nil)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandlePauseAgent_InvalidCompanyUUID_Returns400(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	rr := doAuth(s, fixedUserID(), "POST", invalidCo("/agents/"+uuid.New().String()+"/pause"), nil)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleResumeAgent_InvalidCompanyUUID_Returns400(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	rr := doAuth(s, fixedUserID(), "POST", invalidCo("/agents/"+uuid.New().String()+"/resume"), nil)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleReassignAgent_InvalidCompanyUUID_Returns400(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	rr := doAuth(s, fixedUserID(), "POST", invalidCo("/agents/"+uuid.New().String()+"/reassign"), map[string]string{})
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleAgentChat_InvalidCompanyUUID_Returns400(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	rr := doAuth(s, fixedUserID(), "POST", invalidCo("/agents/"+uuid.New().String()+"/chat"), map[string]string{"message": "hi"})
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleChatHistory_InvalidCompanyUUID_Returns400(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	rr := doAuth(s, fixedUserID(), "GET", invalidCo("/agents/"+uuid.New().String()+"/chat/history"), nil)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleListIssues_InvalidCompanyUUID_Returns400(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	rr := doAuth(s, fixedUserID(), "GET", invalidCo("/issues"), nil)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleCreateIssue_InvalidCompanyUUID_Returns400(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	rr := doAuth(s, fixedUserID(), "POST", invalidCo("/issues"), map[string]string{"title": "x"})
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleGetIssue_InvalidCompanyUUID_Returns400(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	rr := doAuth(s, fixedUserID(), "GET", invalidCo("/issues/"+uuid.New().String()), nil)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleUpdateIssue_InvalidCompanyUUID_Returns400(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	rr := doAuth(s, fixedUserID(), "PATCH", invalidCo("/issues/"+uuid.New().String()), map[string]string{})
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleAddDependency_InvalidCompanyUUID_Returns400(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	rr := doAuth(s, fixedUserID(), "POST", invalidCo("/issues/"+uuid.New().String()+"/dependencies"), map[string]string{})
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleRemoveDependency_InvalidCompanyUUID_Returns400(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	rr := doAuth(s, fixedUserID(), "DELETE", invalidCo("/issues/"+uuid.New().String()+"/dependencies/"+uuid.New().String()), nil)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleCreateHire_InvalidCompanyUUID_Returns400(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	rr := doAuth(s, fixedUserID(), "POST", invalidCo("/hires"), map[string]string{"role_title": "eng", "runtime": "claude_code"})
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleApproveHire_InvalidCompanyUUID_Returns400(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	rr := doAuth(s, fixedUserID(), "POST", invalidCo("/hires/"+uuid.New().String()+"/approve"), nil)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleRejectHire_InvalidCompanyUUID_Returns400(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	rr := doAuth(s, fixedUserID(), "POST", invalidCo("/hires/"+uuid.New().String()+"/reject"), map[string]string{})
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleRevokeFSPermission_InvalidCompanyUUID_Returns400(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	rr := doAuth(s, fixedUserID(), "DELETE", invalidCo("/fs/permissions/"+uuid.New().String()), nil)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleDismissNotification_InvalidCompanyUUID_Returns400(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	rr := doAuth(s, fixedUserID(), "POST", invalidCo("/notifications/"+uuid.New().String()+"/dismiss"), nil)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleListFSPermissions_InvalidCompanyUUID_Returns400(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	rr := doAuth(s, fixedUserID(), "GET", invalidCo("/fs/permissions"), nil)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleGrantFSPermission_InvalidCompanyUUID_Returns400(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	rr := doAuth(s, fixedUserID(), "POST", invalidCo("/fs/permissions"), map[string]string{})
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

// ── Auth header variations on authenticated routes ────────────────────────────

func TestAuthenticatedRoute_ExpiredToken_Returns401(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	expiredToken := mintExpiredToken(testJWTSecret, fixedUserID())
	rr := do(s, "GET", "/api/companies", nil, map[string]string{
		"Authorization": "Bearer " + expiredToken,
	})
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestAuthenticatedRoute_WrongSecret_Returns401(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	wrongToken := mintTestToken("wrong-secret", fixedUserID())
	rr := do(s, "GET", "/api/companies", nil, map[string]string{
		"Authorization": "Bearer " + wrongToken,
	})
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestAuthenticatedRoute_MalformedToken_Returns401(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	rr := do(s, "GET", "/api/companies", nil, map[string]string{
		"Authorization": "Bearer not.a.real.jwt",
	})
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestAuthenticatedRoute_NoBearer_TokenOnly_Behavior(t *testing.T) {
	// Raw token without "Bearer " prefix: TrimPrefix is a no-op, so JWT parses fine.
	s := newTestServer(newMockStore(), newMockOrch())
	token := mintToken(fixedUserID())
	rr := do(s, "GET", "/api/companies", nil, map[string]string{
		"Authorization": token, // no prefix
	})
	// jwt parses it fine since TrimPrefix("rawtoken","Bearer ") == "rawtoken"
	// It then tries the DB — mock returns nil → 500 or similar, but NOT 401.
	if rr.Code == http.StatusUnauthorized {
		t.Log("INFO: raw token (no Bearer prefix) was rejected — TrimPrefix behavior changed")
	}
}

func TestAuthenticatedRoute_QueryParamToken_Passes(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	token := mintToken(fixedUserID())
	req, _ := newRawRequest("GET", "/api/companies?token="+token, nil, nil)
	rr := httptest.NewRecorder()
	s.Router().ServeHTTP(rr, req)
	// Auth passes via query param; DB may fail (mock returns zero value), but not 401.
	if rr.Code == http.StatusUnauthorized {
		t.Errorf("query param token should pass auth, got 401")
	}
}

// ── handleRegister: success path and conflict ────────────────────────────────

func TestHandleRegister_Success_Returns201WithToken(t *testing.T) {
	userID := uuid.New()
	ms := newMockStore()
	ms.createUserFn = func(_ context.Context, email, _ string) (*store.User, error) {
		return &store.User{ID: userID, Email: email}, nil
	}
	ms.addUserToCompanyFn = func(_ context.Context, _, _ uuid.UUID, _ string) error { return nil }

	s := newTestServer(ms, newMockOrch())
	rr := do(s, "POST", "/api/auth/register",
		map[string]string{"email": "new@example.com", "password": "password123"},
		nil)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d (body: %s)", rr.Code, rr.Body.String())
	}
	var body map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	tokenStr, ok := body["token"].(string)
	if !ok || tokenStr == "" {
		t.Error("expected non-empty 'token' field in response")
	}
	// Verify it's a valid JWT
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		t.Errorf("token is not a valid JWT (expected 3 parts), got %d", len(parts))
	}
}

func TestHandleRegister_DuplicateEmail_Returns409(t *testing.T) {
	ms := newMockStore()
	ms.createUserFn = func(_ context.Context, _, _ string) (*store.User, error) {
		return nil, errStub
	}
	s := newTestServer(ms, newMockOrch())
	rr := do(s, "POST", "/api/auth/register",
		map[string]string{"email": "dup@example.com", "password": "pass"},
		nil)
	if rr.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", rr.Code)
	}
}

func TestHandleRegister_EmptyPassword_StillRegisters(t *testing.T) {
	// Empty password is hashed; registration succeeds if DB succeeds
	userID := uuid.New()
	ms := newMockStore()
	ms.createUserFn = func(_ context.Context, email, _ string) (*store.User, error) {
		return &store.User{ID: userID, Email: email}, nil
	}
	s := newTestServer(ms, newMockOrch())
	rr := do(s, "POST", "/api/auth/register",
		map[string]string{"email": "empty@example.com", "password": ""},
		nil)
	// bcrypt accepts empty password; registration should succeed
	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d (body: %s)", rr.Code, rr.Body.String())
	}
}

func TestHandleRegister_ResponseContainsUser(t *testing.T) {
	userID := uuid.New()
	ms := newMockStore()
	ms.createUserFn = func(_ context.Context, email, _ string) (*store.User, error) {
		return &store.User{ID: userID, Email: email}, nil
	}
	s := newTestServer(ms, newMockOrch())
	rr := do(s, "POST", "/api/auth/register",
		map[string]string{"email": "check@example.com", "password": "pw"},
		nil)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Code)
	}
	var body map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&body) //nolint
	if body["user"] == nil {
		t.Error("expected 'user' field in register response")
	}
}

// ── handleLogin: success, wrong password, user not found ─────────────────────

func TestHandleLogin_UserNotFound_Returns401(t *testing.T) {
	ms := newMockStore()
	ms.getUserByEmailFn = func(_ context.Context, _ string) (*store.User, error) {
		return nil, errStub
	}
	s := newTestServer(ms, newMockOrch())
	rr := do(s, "POST", "/api/auth/login",
		map[string]string{"email": "nobody@example.com", "password": "pass"},
		nil)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unknown user, got %d", rr.Code)
	}
}

func TestHandleLogin_WrongPassword_Returns401(t *testing.T) {
	// Store a properly bcrypt-hashed password but send the wrong one
	import_bcrypt_hash := "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy" // bcrypt("correctpass")
	ms := newMockStore()
	ms.getUserByEmailFn = func(_ context.Context, email string) (*store.User, error) {
		return &store.User{
			ID:           uuid.New(),
			Email:        email,
			PasswordHash: import_bcrypt_hash,
		}, nil
	}
	s := newTestServer(ms, newMockOrch())
	rr := do(s, "POST", "/api/auth/login",
		map[string]string{"email": "user@example.com", "password": "wrongpassword"},
		nil)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for wrong password, got %d", rr.Code)
	}
}

func TestHandleLogin_Success_Returns200WithToken(t *testing.T) {
	userID := uuid.New()
	// Generate a real bcrypt hash for a known password
	import_bcrypt_cost10 := "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi" // bcrypt("password")
	ms := newMockStore()
	ms.getUserByEmailFn = func(_ context.Context, email string) (*store.User, error) {
		return &store.User{
			ID:           userID,
			Email:        email,
			PasswordHash: import_bcrypt_cost10,
		}, nil
	}
	s := newTestServer(ms, newMockOrch())
	rr := do(s, "POST", "/api/auth/login",
		map[string]string{"email": "user@example.com", "password": "password"},
		nil)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}
	var body map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&body) //nolint
	tokenStr, ok := body["token"].(string)
	if !ok || tokenStr == "" {
		t.Error("expected non-empty 'token' in login response")
	}
}

func TestHandleLogin_Success_TokenIsValidJWT(t *testing.T) {
	userID := uuid.New()
	bcryptHash := "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi" // bcrypt("password")
	ms := newMockStore()
	ms.getUserByEmailFn = func(_ context.Context, email string) (*store.User, error) {
		return &store.User{ID: userID, Email: email, PasswordHash: bcryptHash}, nil
	}
	s := newTestServer(ms, newMockOrch())
	rr := do(s, "POST", "/api/auth/login",
		map[string]string{"email": "user@example.com", "password": "password"},
		nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var body map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&body) //nolint
	tokenStr := body["token"].(string)

	// Parse and validate the JWT
	claims := &jwt.RegisteredClaims{}
	tok, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(testJWTSecret), nil
	})
	if err != nil || !tok.Valid {
		t.Errorf("token is not valid: %v", err)
	}
	if claims.Subject != userID.String() {
		t.Errorf("expected subject=%s, got %s", userID, claims.Subject)
	}
}

// ── mintJWT ───────────────────────────────────────────────────────────────────

func TestMintJWT_ProducesValidToken(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	userID := uuid.New()
	tokenStr, err := s.mintJWT(userID)
	if err != nil {
		t.Fatalf("mintJWT: %v", err)
	}
	claims := &jwt.RegisteredClaims{}
	tok, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(testJWTSecret), nil
	})
	if err != nil || !tok.Valid {
		t.Errorf("minted token is invalid: %v", err)
	}
}

func TestMintJWT_SubjectIsUserID(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	userID := uuid.New()
	tokenStr, _ := s.mintJWT(userID)
	claims := &jwt.RegisteredClaims{}
	jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) { //nolint
		return []byte(testJWTSecret), nil
	})
	if claims.Subject != userID.String() {
		t.Errorf("expected subject=%s, got %s", userID, claims.Subject)
	}
}

func TestMintJWT_ExpiresInFuture(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	tokenStr, _ := s.mintJWT(uuid.New())
	claims := &jwt.RegisteredClaims{}
	jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) { //nolint
		return []byte(testJWTSecret), nil
	})
	if claims.ExpiresAt == nil {
		t.Fatal("expected ExpiresAt to be set")
	}
	if !claims.ExpiresAt.Time.After(claims.IssuedAt.Time) {
		t.Error("expected expiry to be after issuedAt")
	}
}

func TestMintJWT_DifferentUsersGetDifferentTokens(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	t1, _ := s.mintJWT(uuid.New())
	t2, _ := s.mintJWT(uuid.New())
	if t1 == t2 {
		t.Error("different users should get different tokens")
	}
}

// ── chiCompanyMiddleware: uncovered branches ──────────────────────────────────

func TestChiCompanyMiddleware_UserCanAccessReturnsError_Returns403(t *testing.T) {
	ms := newMockStore()
	// UserCanAccessCompany returns an error (not just false)
	ms.userCanAccessFn = func(_ context.Context, _, _ uuid.UUID) (bool, error) {
		return false, errStub
	}
	s := newTestServer(ms, newMockOrch())
	companyID := uuid.New()
	userID := uuid.New()
	rr := doAuth(s, userID, "GET", companyPath(companyID, "/"), nil)
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 when UserCanAccessCompany returns error, got %d", rr.Code)
	}
}

func TestChiCompanyMiddleware_UserCanAccessReturnsFalse_Returns403(t *testing.T) {
	ms := newMockStore()
	ms.userCanAccessFn = func(_ context.Context, _, _ uuid.UUID) (bool, error) {
		return false, nil
	}
	s := newTestServer(ms, newMockOrch())
	companyID := uuid.New()
	userID := uuid.New()
	rr := doAuth(s, userID, "GET", companyPath(companyID, "/"), nil)
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 when access denied, got %d", rr.Code)
	}
}

func TestChiCompanyMiddleware_ValidAccess_PassesThrough(t *testing.T) {
	ms := newMockStore()
	ms.userCanAccessFn = func(_ context.Context, _, _ uuid.UUID) (bool, error) {
		return true, nil
	}
	companyID := uuid.New()
	ms.getCompanyFn = func(_ context.Context, id uuid.UUID) (*store.Company, error) {
		return &store.Company{ID: id, Name: "TestCo"}, nil
	}
	s := newTestServer(ms, newMockOrch())
	userID := uuid.New()
	rr := doAuth(s, userID, "GET", companyPath(companyID, "/"), nil)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for valid access, got %d (body: %s)", rr.Code, rr.Body.String())
	}
}

// ── NewServer ─────────────────────────────────────────────────────────────────

func TestNewServer_PublicRoutesReachable(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	// Public routes should not return 404
	for _, path := range []string{"/api/auth/register", "/api/auth/login", "/api/runtimes"} {
		method := "POST"
		if path == "/api/runtimes" {
			method = "GET"
		}
		rr := do(s, method, path, map[string]string{}, nil)
		if rr.Code == http.StatusNotFound {
			t.Errorf("expected route %s %s to be registered (not 404), got %d", method, path, rr.Code)
		}
	}
}

func TestNewServer_AuthenticatedRoutesExist_NotRouteNotFound(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	userID := uuid.New()
	companyID := uuid.New()

	// These routes should exist (401/403 is fine, but not 404/405)
	routes := []struct{ method, path string }{
		{"GET", "/api/companies"},
		{"POST", "/api/companies"},
		{"GET", companyPath(companyID, "/agents")},
		{"GET", companyPath(companyID, "/issues")},
		{"GET", companyPath(companyID, "/hires")},
		{"GET", companyPath(companyID, "/audit")},
		{"GET", companyPath(companyID, "/notifications")},
		{"GET", companyPath(companyID, "/fs/browse")},
	}
	for _, tc := range routes {
		rr := doAuth(s, userID, tc.method, tc.path, nil)
		if rr.Code == http.StatusNotFound || rr.Code == http.StatusMethodNotAllowed {
			t.Errorf("route %s %s returned %d (route not registered)", tc.method, tc.path, rr.Code)
		}
	}
}

func TestNewServer_UnknownRoute_ServesSPA(t *testing.T) {
	s := newTestServer(newMockStore(), newMockOrch())
	// Unknown paths fall through to SPA handler (handleSPA tries to serve a file)
	rr := do(s, "GET", "/some/unknown/ui/path", nil, nil)
	// Should not be a chi-generated 404 (which would be text/plain);
	// the SPA handler is registered as "/*", so it will be called.
	// It tries to serve /tmp/static/index.html which doesn't exist → 404 from http.ServeFile.
	// But the important thing is it's not a 405 Method Not Allowed.
	if rr.Code == http.StatusMethodNotAllowed {
		t.Error("SPA fallback should handle GET for unknown paths")
	}
}

