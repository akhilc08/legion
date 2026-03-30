package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// base64URLEncode encodes bytes using URL-safe base64 without padding.
func base64URLEncode(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func makeTestServer() *Server {
	return &Server{jwtSecret: "test-secret"}
}

func mintTestToken(secret string, userID uuid.UUID) string {
	claims := jwt.RegisteredClaims{
		Subject:   userID.String(),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}
	tok, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	return tok
}

func mintExpiredToken(secret string, userID uuid.UUID) string {
	claims := jwt.RegisteredClaims{
		Subject:   userID.String(),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
	}
	tok, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	return tok
}

func mintTokenWithSubject(secret, subject string) string {
	claims := jwt.RegisteredClaims{
		Subject:   subject,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}
	tok, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	return tok
}

// okHandler records whether it was called.
func okHandler(called *bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*called = true
		w.WriteHeader(http.StatusOK)
	})
}

// ── authMiddleware ────────────────────────────────────────────────────────────

func TestAuthMiddleware_NoToken_Returns401(t *testing.T) {
	s := makeTestServer()
	called := false
	handler := s.authMiddleware(okHandler(&called))
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if called {
		t.Fatal("handler should not be called without token")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestAuthMiddleware_BearerHeader_Returns200(t *testing.T) {
	s := makeTestServer()
	userID := uuid.New()
	token := mintTestToken("test-secret", userID)
	called := false
	handler := s.authMiddleware(okHandler(&called))
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if !called {
		t.Fatal("handler was not called")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestAuthMiddleware_QueryParam_Returns200(t *testing.T) {
	s := makeTestServer()
	userID := uuid.New()
	token := mintTestToken("test-secret", userID)
	called := false
	handler := s.authMiddleware(okHandler(&called))
	req := httptest.NewRequest("GET", "/?token="+token, nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if !called {
		t.Fatal("handler was not called")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestAuthMiddleware_HeaderTakesPrecedenceOverQuery(t *testing.T) {
	s := makeTestServer()
	// Header has valid token; query has token signed with wrong secret (would fail alone).
	validUserID := uuid.New()
	validToken := mintTestToken("test-secret", validUserID)
	wrongToken := mintTestToken("wrong-secret", uuid.New())

	var capturedID uuid.UUID
	handler := s.authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedID, _ = userIDFromCtx(r)
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/?token="+wrongToken, nil)
	req.Header.Set("Authorization", "Bearer "+validToken)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 (header takes precedence), got %d", rr.Code)
	}
	if capturedID != validUserID {
		t.Errorf("expected userID from header token, got %s", capturedID)
	}
}

func TestAuthMiddleware_ExpiredToken_Returns401(t *testing.T) {
	s := makeTestServer()
	userID := uuid.New()
	token := mintExpiredToken("test-secret", userID)
	called := false
	handler := s.authMiddleware(okHandler(&called))
	req := httptest.NewRequest("GET", "/?token="+token, nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if called {
		t.Fatal("handler should not be called with expired token")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestAuthMiddleware_WrongSecret_Returns401(t *testing.T) {
	s := makeTestServer()
	userID := uuid.New()
	token := mintTestToken("wrong-secret", userID)
	called := false
	handler := s.authMiddleware(okHandler(&called))
	req := httptest.NewRequest("GET", "/?token="+token, nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if called {
		t.Fatal("handler should not be called with wrong secret")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestAuthMiddleware_MalformedToken_Returns401(t *testing.T) {
	s := makeTestServer()
	called := false
	handler := s.authMiddleware(okHandler(&called))
	req := httptest.NewRequest("GET", "/?token=not.a.valid.jwt", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if called {
		t.Fatal("handler should not be called with malformed token")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestAuthMiddleware_TwoSegments_Returns401(t *testing.T) {
	s := makeTestServer()
	called := false
	handler := s.authMiddleware(okHandler(&called))
	// Only 2 segments — not a valid JWT
	req := httptest.NewRequest("GET", "/?token=header.payload", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if called {
		t.Fatal("handler should not be called")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestAuthMiddleware_InvalidUUIDSubject_Returns401(t *testing.T) {
	s := makeTestServer()
	// Subject is not a UUID
	token := mintTokenWithSubject("test-secret", "not-a-uuid")
	called := false
	handler := s.authMiddleware(okHandler(&called))
	req := httptest.NewRequest("GET", "/?token="+token, nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if called {
		t.Fatal("handler should not be called with invalid UUID subject")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestAuthMiddleware_EmptySubject_Returns401(t *testing.T) {
	s := makeTestServer()
	token := mintTokenWithSubject("test-secret", "")
	called := false
	handler := s.authMiddleware(okHandler(&called))
	req := httptest.NewRequest("GET", "/?token="+token, nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if called {
		t.Fatal("handler should not be called with empty subject")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestAuthMiddleware_SetsUserIDInContext(t *testing.T) {
	s := makeTestServer()
	userID := uuid.New()
	token := mintTestToken("test-secret", userID)
	var gotID uuid.UUID
	var gotOK bool
	handler := s.authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotID, gotOK = userIDFromCtx(r)
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/?token="+token, nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if !gotOK {
		t.Fatal("userID not found in context")
	}
	if gotID != userID {
		t.Errorf("expected %s, got %s", userID, gotID)
	}
}

func TestAuthMiddleware_ValidTokenCallsNext(t *testing.T) {
	s := makeTestServer()
	userID := uuid.New()
	token := mintTestToken("test-secret", userID)
	called := false
	handler := s.authMiddleware(okHandler(&called))
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if !called {
		t.Fatal("next handler was not called")
	}
}

func TestAuthMiddleware_TamperedSignature_Returns401(t *testing.T) {
	s := makeTestServer()
	userID := uuid.New()
	token := mintTestToken("test-secret", userID)
	// Replace last few chars of the signature to tamper it
	parts := strings.Split(token, ".")
	if len(parts) == 3 {
		sig := []rune(parts[2])
		// Flip the last character
		if sig[len(sig)-1] == 'a' {
			sig[len(sig)-1] = 'b'
		} else {
			sig[len(sig)-1] = 'a'
		}
		parts[2] = string(sig)
	}
	tamperedToken := strings.Join(parts, ".")
	called := false
	handler := s.authMiddleware(okHandler(&called))
	req := httptest.NewRequest("GET", "/?token="+tamperedToken, nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if called {
		t.Fatal("handler should not be called with tampered token")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestAuthMiddleware_LowercaseBearerPrefix_Behavior(t *testing.T) {
	// "bearer " (lowercase) is NOT stripped — TrimPrefix is case-sensitive.
	// So the token string will include "bearer " prefix and fail JWT parsing.
	s := makeTestServer()
	userID := uuid.New()
	token := mintTestToken("test-secret", userID)
	called := false
	handler := s.authMiddleware(okHandler(&called))
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "bearer "+token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	// The current implementation does NOT lowercase-strip "bearer ", so this fails.
	if called {
		t.Log("INFO: lowercase 'bearer' prefix was accepted (case-insensitive behavior)")
	} else {
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 for lowercase bearer prefix, got %d", rr.Code)
		}
		t.Log("INFO: lowercase 'bearer' prefix was rejected (case-sensitive behavior — documented)")
	}
}

func TestAuthMiddleware_RawTokenNoPrefix_Behavior(t *testing.T) {
	// Sending the raw JWT in Authorization header without "Bearer " prefix.
	// TrimPrefix("rawtoken", "Bearer ") == "rawtoken" (unchanged), so it parses fine.
	s := makeTestServer()
	userID := uuid.New()
	token := mintTestToken("test-secret", userID)
	called := false
	handler := s.authMiddleware(okHandler(&called))
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", token) // no "Bearer " prefix
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	// TrimPrefix leaves the token unchanged when prefix is absent, so JWT parses fine.
	if !called {
		t.Logf("INFO: raw token without 'Bearer' prefix was rejected (status %d)", rr.Code)
	} else {
		t.Log("INFO: raw token without 'Bearer' prefix was accepted (TrimPrefix no-op)")
	}
}

func TestAuthMiddleware_VeryLongToken_Returns401(t *testing.T) {
	s := makeTestServer()
	longToken := strings.Repeat("a", 10000)
	called := false
	handler := s.authMiddleware(okHandler(&called))
	req := httptest.NewRequest("GET", "/?token="+longToken, nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if called {
		t.Fatal("handler should not be called with very long nonsense token")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestAuthMiddleware_FutureIssuedAt_Behavior(t *testing.T) {
	// Token with IssuedAt in the future — jwt/v5 does not reject future iat by default.
	s := makeTestServer()
	userID := uuid.New()
	claims := jwt.RegisteredClaims{
		Subject:   userID.String(),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now().Add(time.Hour)), // future
	}
	tok, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("test-secret"))
	called := false
	handler := s.authMiddleware(okHandler(&called))
	req := httptest.NewRequest("GET", "/?token="+tok, nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	// Document the actual behavior; jwt/v5 doesn't reject future iat.
	if called {
		t.Log("INFO: token with future IssuedAt was accepted (jwt/v5 does not validate iat)")
	} else {
		t.Logf("INFO: token with future IssuedAt was rejected (status %d)", rr.Code)
	}
}

func TestAuthMiddleware_AlgNone_Returns401(t *testing.T) {
	// Security: "alg: none" tokens must be rejected.
	// We construct a token manually using the "none" algorithm.
	s := makeTestServer()
	userID := uuid.New()
	// Build a "none" algorithm token by hand: header.payload.
	header := `{"alg":"none","typ":"JWT"}`
	payload := `{"sub":"` + userID.String() + `","exp":` + strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10) + `}`

	headerEnc := base64URLEncode([]byte(header))
	payloadEnc := base64URLEncode([]byte(payload))
	noneToken := headerEnc + "." + payloadEnc + "."

	called := false
	handler := s.authMiddleware(okHandler(&called))
	req := httptest.NewRequest("GET", "/?token="+noneToken, nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if called {
		t.Error("SECURITY BUG: alg=none token was accepted")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for alg=none token, got %d", rr.Code)
	}
}

func TestAuthMiddleware_HeaderStillWorks(t *testing.T) {
	s := makeTestServer()
	userID := uuid.New()
	token := mintTestToken("test-secret", userID)
	called := false
	handler := s.authMiddleware(okHandler(&called))
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if !called {
		t.Fatal("handler was not called")
	}
}

// ── companyAccessMiddleware ───────────────────────────────────────────────────

func TestCompanyAccessMiddleware_NoUserIDInContext_Returns401(t *testing.T) {
	s := makeTestServer()
	called := false
	handler := s.companyAccessMiddleware(okHandler(&called))
	// Request with no user in context
	req := httptest.NewRequest("GET", "/api/companies/"+uuid.New().String()+"/agents", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if called {
		t.Fatal("handler should not be called without userID in context")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestCompanyAccessMiddleware_InvalidCompanyUUID_Returns400(t *testing.T) {
	s := makeTestServer()
	userID := uuid.New()
	called := false
	handler := s.companyAccessMiddleware(okHandler(&called))
	req := httptest.NewRequest("GET", "/api/companies/not-a-uuid/agents", nil)
	// Inject userID into context directly
	ctx := context.WithValue(req.Context(), ctxUserID, userID)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if called {
		t.Fatal("handler should not be called with invalid company UUID")
	}
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestCompanyAccessMiddleware_PathParsing_CompaniesSegment(t *testing.T) {
	s := makeTestServer()
	userID := uuid.New()
	// Use a valid UUID; DB is nil so it will panic past UUID validation — we only test UUID parse.
	called := false
	handler := s.companyAccessMiddleware(okHandler(&called))
	req := httptest.NewRequest("GET", "/api/companies/not-a-uuid/agents", nil)
	ctx := context.WithValue(req.Context(), ctxUserID, userID)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid UUID in companies path, got %d", rr.Code)
	}
}

func TestCompanyAccessMiddleware_MalformedPath_NoCompaniesSegment_Returns400(t *testing.T) {
	s := makeTestServer()
	userID := uuid.New()
	called := false
	handler := s.companyAccessMiddleware(okHandler(&called))
	// Path has no "companies" segment — companyIDStr will be empty, UUID parse fails → 400
	req := httptest.NewRequest("GET", "/api/orgs/"+uuid.New().String()+"/agents", nil)
	ctx := context.WithValue(req.Context(), ctxUserID, userID)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if called {
		t.Fatal("handler should not be called with malformed path")
	}
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for path without 'companies' segment, got %d", rr.Code)
	}
}

func TestCompanyAccessMiddleware_PathWithTrailingSlash(t *testing.T) {
	s := makeTestServer()
	userID := uuid.New()
	called := false
	handler := s.companyAccessMiddleware(okHandler(&called))
	// Trailing slash after UUID — UUID extraction should still work (UUID is at i+1)
	req := httptest.NewRequest("GET", "/api/companies/not-a-uuid/", nil)
	ctx := context.WithValue(req.Context(), ctxUserID, userID)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if called {
		t.Fatal("should not be called")
	}
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestCompanyAccessMiddleware_EmptyPathAfterCompanies_Returns400(t *testing.T) {
	s := makeTestServer()
	userID := uuid.New()
	called := false
	handler := s.companyAccessMiddleware(okHandler(&called))
	// "companies" is the last segment — no segment after it
	req := httptest.NewRequest("GET", "/api/companies", nil)
	ctx := context.WithValue(req.Context(), ctxUserID, userID)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if called {
		t.Fatal("handler should not be called")
	}
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 when companies is last segment, got %d", rr.Code)
	}
}

// ── writeJSON helper ──────────────────────────────────────────────────────────

func TestWriteJSON_SetsContentType(t *testing.T) {
	rr := httptest.NewRecorder()
	writeJSON(rr, http.StatusOK, map[string]string{"key": "value"})
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}
}

func TestWriteJSON_SetsStatusCode(t *testing.T) {
	rr := httptest.NewRecorder()
	writeJSON(rr, http.StatusCreated, map[string]string{})
	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", rr.Code)
	}
}

func TestWriteJSON_ValidBody(t *testing.T) {
	rr := httptest.NewRecorder()
	writeJSON(rr, http.StatusOK, map[string]string{"hello": "world"})
	var result map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("body is not valid JSON: %v", err)
	}
	if result["hello"] != "world" {
		t.Errorf("expected 'world', got %q", result["hello"])
	}
}

func TestWriteJSON_NilValue(t *testing.T) {
	rr := httptest.NewRecorder()
	writeJSON(rr, http.StatusOK, nil)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestWriteJSON_400Status(t *testing.T) {
	rr := httptest.NewRecorder()
	writeJSON(rr, http.StatusBadRequest, map[string]string{"error": "bad"})
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

// ── decodeJSON helper ─────────────────────────────────────────────────────────

func TestDecodeJSON_ValidBody(t *testing.T) {
	body := `{"name":"test"}`
	req := httptest.NewRequest("POST", "/", strings.NewReader(body))
	var result struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(req, &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "test" {
		t.Errorf("expected 'test', got %q", result.Name)
	}
}

func TestDecodeJSON_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest("POST", "/", strings.NewReader("not json {{{"))
	var result map[string]string
	if err := decodeJSON(req, &result); err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestDecodeJSON_EmptyBody(t *testing.T) {
	req := httptest.NewRequest("POST", "/", strings.NewReader(""))
	var result map[string]string
	if err := decodeJSON(req, &result); err == nil {
		t.Fatal("expected error for empty body, got nil")
	}
}

func TestDecodeJSON_ExtraFields(t *testing.T) {
	// json.Decoder ignores unknown fields by default
	body := `{"name":"test","extra":"ignored"}`
	req := httptest.NewRequest("POST", "/", strings.NewReader(body))
	var result struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(req, &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "test" {
		t.Errorf("expected 'test', got %q", result.Name)
	}
}

// ── userIDFromCtx ─────────────────────────────────────────────────────────────

func TestUserIDFromCtx_Present(t *testing.T) {
	id := uuid.New()
	req := httptest.NewRequest("GET", "/", nil)
	ctx := context.WithValue(req.Context(), ctxUserID, id)
	req = req.WithContext(ctx)
	got, ok := userIDFromCtx(req)
	if !ok {
		t.Fatal("expected ok=true, got false")
	}
	if got != id {
		t.Errorf("expected %s, got %s", id, got)
	}
}

func TestUserIDFromCtx_Absent(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	_, ok := userIDFromCtx(req)
	if ok {
		t.Fatal("expected ok=false when no userID in context")
	}
}

func TestUserIDFromCtx_WrongType(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	ctx := context.WithValue(req.Context(), ctxUserID, "not-a-uuid")
	req = req.WithContext(ctx)
	_, ok := userIDFromCtx(req)
	if ok {
		t.Fatal("expected ok=false when wrong type in context")
	}
}

// ── companyIDFromCtx ──────────────────────────────────────────────────────────

func TestCompanyIDFromCtx_Present(t *testing.T) {
	id := uuid.New()
	req := httptest.NewRequest("GET", "/", nil)
	ctx := context.WithValue(req.Context(), ctxCompanyID, id)
	req = req.WithContext(ctx)
	got, ok := companyIDFromCtx(req)
	if !ok {
		t.Fatal("expected ok=true, got false")
	}
	if got != id {
		t.Errorf("expected %s, got %s", id, got)
	}
}

func TestCompanyIDFromCtx_Absent(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	_, ok := companyIDFromCtx(req)
	if ok {
		t.Fatal("expected ok=false when no companyID in context")
	}
}

func TestCompanyIDFromCtx_WrongType(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	ctx := context.WithValue(req.Context(), ctxCompanyID, 12345)
	req = req.WithContext(ctx)
	_, ok := companyIDFromCtx(req)
	if ok {
		t.Fatal("expected ok=false when wrong type in context")
	}
}

// ── error response body structure ─────────────────────────────────────────────

func TestAuthMiddleware_NoToken_ErrorBody(t *testing.T) {
	s := makeTestServer()
	handler := s.authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	var body map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
	if body["error"] == "" {
		t.Error("expected non-empty 'error' field in response body")
	}
}

func TestAuthMiddleware_WrongSecret_ErrorBody(t *testing.T) {
	s := makeTestServer()
	token := mintTestToken("wrong", uuid.New())
	handler := s.authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	req := httptest.NewRequest("GET", "/?token="+token, nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	var body map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
	if body["error"] != "unauthorized" {
		t.Errorf("expected error='unauthorized', got %q", body["error"])
	}
}
