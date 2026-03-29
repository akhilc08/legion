package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

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

func TestAuthMiddleware_QueryParam(t *testing.T) {
	s := makeTestServer()
	userID := uuid.New()
	token := mintTestToken("test-secret", userID)

	called := false
	handler := s.authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		gotID, ok := userIDFromCtx(r)
		if !ok || gotID != userID {
			t.Errorf("expected userID %s in context, got %s (ok=%v)", userID, gotID, ok)
		}
	}))

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

func TestAuthMiddleware_HeaderStillWorks(t *testing.T) {
	s := makeTestServer()
	userID := uuid.New()
	token := mintTestToken("test-secret", userID)

	called := false
	handler := s.authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !called {
		t.Fatal("handler was not called")
	}
}
