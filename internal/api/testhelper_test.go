package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"time"

	"conductor/internal/ws"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const testJWTSecret = "test-secret"

// newTestServer builds a Server wired with the provided mock store and
// orchestrator, using a fixed JWT secret.
func newTestServer(db Store, orch Orchestrator) *Server {
	hub := ws.NewHub()
	return NewServer(db, hub, orch, testJWTSecret, "/tmp/static")
}

// mintToken creates a valid JWT for the given userID.
func mintToken(userID uuid.UUID) string {
	claims := jwt.RegisteredClaims{
		Subject:   userID.String(),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	tok, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testJWTSecret))
	return tok
}

// authHeader returns a Bearer authorization header value.
func authHeader(userID uuid.UUID) string {
	return "Bearer " + mintToken(userID)
}

// do executes a request against srv.Router() and returns the recorder.
func do(srv *Server, method, path string, body interface{}, headers map[string]string) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body) //nolint
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rr := httptest.NewRecorder()
	srv.Router().ServeHTTP(rr, req)
	return rr
}

// doAuth executes an authenticated request for the given userID.
func doAuth(srv *Server, userID uuid.UUID, method, path string, body interface{}) *httptest.ResponseRecorder {
	return do(srv, method, path, body, map[string]string{
		"Authorization": authHeader(userID),
	})
}

// decodeBodyInto decodes the JSON response body into v.
func decodeBodyInto(rr *httptest.ResponseRecorder, v interface{}) error {
	return json.NewDecoder(rr.Body).Decode(v)
}

// decodeBody decodes the JSON response body into a map, fataling on error.
// This signature (t, rr) matches usage in audit/fs/agents/issues test files.
func decodeBody(t interface {
	Helper()
	Fatalf(string, ...interface{})
}, rr *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&m); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	return m
}

// companyPath builds a URL for a company-scoped resource.
func companyPath(companyID uuid.UUID, suffix string) string {
	return fmt.Sprintf("/api/companies/%s%s", companyID, suffix)
}

// setupCompanyAccess configures the mockStore so that UserCanAccessCompany
// always returns true, allowing the chiCompanyMiddleware to pass.
func setupCompanyAccess(ms *mockStore) {
	ms.userCanAccessFn = func(_ context.Context, _, _ uuid.UUID) (bool, error) {
		return true, nil
	}
}
