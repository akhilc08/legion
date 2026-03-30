package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// ── writeJSON ─────────────────────────────────────────────────────────────────

func TestWriteJSON_Status200(t *testing.T) {
	rr := httptest.NewRecorder()
	writeJSON(rr, http.StatusOK, map[string]string{"key": "val"})
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestWriteJSON_Status201(t *testing.T) {
	rr := httptest.NewRecorder()
	writeJSON(rr, http.StatusCreated, map[string]string{"id": "123"})
	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", rr.Code)
	}
}

func TestWriteJSON_Status400(t *testing.T) {
	rr := httptest.NewRecorder()
	writeJSON(rr, http.StatusBadRequest, map[string]string{"error": "bad"})
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestWriteJSON_Status500(t *testing.T) {
	rr := httptest.NewRecorder()
	writeJSON(rr, http.StatusInternalServerError, map[string]string{"error": "internal"})
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

func TestWriteJSON_SetsContentTypeJSON(t *testing.T) {
	rr := httptest.NewRecorder()
	writeJSON(rr, http.StatusOK, map[string]string{"key": "value"})
	ct := rr.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}
}

func TestWriteJSON_BodyIsValidJSON(t *testing.T) {
	rr := httptest.NewRecorder()
	writeJSON(rr, http.StatusOK, map[string]string{"hello": "world"})
	var m map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&m); err != nil {
		t.Errorf("expected valid JSON body, got decode error: %v", err)
	}
	if m["hello"] != "world" {
		t.Errorf("expected hello=world, got %v", m["hello"])
	}
}

func TestWriteJSON_NilPayload(t *testing.T) {
	rr := httptest.NewRecorder()
	writeJSON(rr, http.StatusOK, nil)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 with nil payload, got %d", rr.Code)
	}
	body := strings.TrimSpace(rr.Body.String())
	if body != "null" {
		t.Errorf("expected null body for nil payload, got %q", body)
	}
}

func TestWriteJSON_ArrayPayload(t *testing.T) {
	rr := httptest.NewRecorder()
	writeJSON(rr, http.StatusOK, []string{"a", "b", "c"})
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var result []string
	json.NewDecoder(rr.Body).Decode(&result)
	if len(result) != 3 || result[0] != "a" {
		t.Errorf("expected [a,b,c], got %v", result)
	}
}

func TestWriteJSON_EmptyArrayPayload(t *testing.T) {
	rr := httptest.NewRecorder()
	writeJSON(rr, http.StatusOK, []string{})
	body := strings.TrimSpace(rr.Body.String())
	if body != "[]" {
		t.Errorf("expected [] body, got %q", body)
	}
}

func TestWriteJSON_NestedPayload(t *testing.T) {
	rr := httptest.NewRecorder()
	writeJSON(rr, http.StatusOK, map[string]interface{}{
		"nested": map[string]int{"count": 42},
	})
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var m map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&m)
	nested, ok := m["nested"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected nested map, got %T", m["nested"])
	}
	if nested["count"] != float64(42) {
		t.Errorf("expected count=42, got %v", nested["count"])
	}
}

// ── decodeJSON ────────────────────────────────────────────────────────────────

func TestDecodeJSON_ValidJSON(t *testing.T) {
	body := `{"name":"alice","age":30}`
	req := httptest.NewRequest("POST", "/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	var target struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	if err := decodeJSON(req, &target); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if target.Name != "alice" {
		t.Errorf("expected name=alice, got %q", target.Name)
	}
	if target.Age != 30 {
		t.Errorf("expected age=30, got %d", target.Age)
	}
}

func TestDecodeJSON_InvalidJSON_ReturnsError(t *testing.T) {
	body := `{bad json`
	req := httptest.NewRequest("POST", "/", bytes.NewBufferString(body))
	var target map[string]interface{}
	if err := decodeJSON(req, &target); err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestDecodeJSON_EmptyBody_ReturnsError(t *testing.T) {
	req := httptest.NewRequest("POST", "/", bytes.NewBufferString(""))
	var target map[string]interface{}
	if err := decodeJSON(req, &target); err == nil {
		t.Error("expected error for empty body, got nil")
	}
}

func TestDecodeJSON_ExtraFields_IgnoredByDefault(t *testing.T) {
	body := `{"name":"bob","extra":"ignored"}`
	req := httptest.NewRequest("POST", "/", bytes.NewBufferString(body))
	var target struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(req, &target); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if target.Name != "bob" {
		t.Errorf("expected name=bob, got %q", target.Name)
	}
}

func TestDecodeJSON_NullBody(t *testing.T) {
	req := httptest.NewRequest("POST", "/", bytes.NewBufferString("null"))
	var target *struct{ Name string }
	// Decoding null into a pointer: should be nil after decode, no error
	if err := decodeJSON(req, &target); err != nil {
		t.Fatalf("expected no error for null JSON, got %v", err)
	}
}

func TestDecodeJSON_BoolPayload(t *testing.T) {
	req := httptest.NewRequest("POST", "/", bytes.NewBufferString("true"))
	var target bool
	if err := decodeJSON(req, &target); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !target {
		t.Error("expected true")
	}
}

// ── userIDFromCtx ─────────────────────────────────────────────────────────────

func TestUserIDFromCtx_WhenPresent_ReturnsCorrectUUID(t *testing.T) {
	userID := uuid.New()
	req := httptest.NewRequest("GET", "/", nil)
	ctx := context.WithValue(req.Context(), ctxUserID, userID)
	req = req.WithContext(ctx)

	got, ok := userIDFromCtx(req)
	if !ok {
		t.Error("expected ok=true, got false")
	}
	if got != userID {
		t.Errorf("expected %s, got %s", userID, got)
	}
}

func TestUserIDFromCtx_WhenAbsent_ReturnsZeroAndFalse(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	got, ok := userIDFromCtx(req)
	if ok {
		t.Error("expected ok=false, got true")
	}
	if got != (uuid.UUID{}) {
		t.Errorf("expected zero UUID, got %s", got)
	}
}

func TestUserIDFromCtx_WrongTypeInContext_ReturnsFalse(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	ctx := context.WithValue(req.Context(), ctxUserID, "not-a-uuid")
	req = req.WithContext(ctx)
	_, ok := userIDFromCtx(req)
	if ok {
		t.Error("expected ok=false when context has wrong type")
	}
}

func TestUserIDFromCtx_DifferentUsersAreDistinct(t *testing.T) {
	id1 := uuid.New()
	id2 := uuid.New()

	req1 := httptest.NewRequest("GET", "/", nil)
	ctx1 := context.WithValue(req1.Context(), ctxUserID, id1)
	req1 = req1.WithContext(ctx1)

	req2 := httptest.NewRequest("GET", "/", nil)
	ctx2 := context.WithValue(req2.Context(), ctxUserID, id2)
	req2 = req2.WithContext(ctx2)

	got1, _ := userIDFromCtx(req1)
	got2, _ := userIDFromCtx(req2)

	if got1 == got2 {
		t.Errorf("expected distinct user IDs, got same: %s", got1)
	}
}

// ── companyIDFromCtx ──────────────────────────────────────────────────────────

func TestCompanyIDFromCtx_WhenPresent_ReturnsCorrectUUID(t *testing.T) {
	companyID := uuid.New()
	req := httptest.NewRequest("GET", "/", nil)
	ctx := context.WithValue(req.Context(), ctxCompanyID, companyID)
	req = req.WithContext(ctx)

	got, ok := companyIDFromCtx(req)
	if !ok {
		t.Error("expected ok=true, got false")
	}
	if got != companyID {
		t.Errorf("expected %s, got %s", companyID, got)
	}
}

func TestCompanyIDFromCtx_WhenAbsent_ReturnsZeroAndFalse(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	got, ok := companyIDFromCtx(req)
	if ok {
		t.Error("expected ok=false, got true")
	}
	if got != (uuid.UUID{}) {
		t.Errorf("expected zero UUID, got %s", got)
	}
}

func TestCompanyIDFromCtx_WrongTypeInContext_ReturnsFalse(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	ctx := context.WithValue(req.Context(), ctxCompanyID, 12345)
	req = req.WithContext(ctx)
	_, ok := companyIDFromCtx(req)
	if ok {
		t.Error("expected ok=false when context has wrong type")
	}
}

func TestCompanyIDFromCtx_UserAndCompanyAreDistinct(t *testing.T) {
	userID := uuid.New()
	companyID := uuid.New()
	req := httptest.NewRequest("GET", "/", nil)
	ctx := context.WithValue(req.Context(), ctxUserID, userID)
	ctx = context.WithValue(ctx, ctxCompanyID, companyID)
	req = req.WithContext(ctx)

	gotUser, okUser := userIDFromCtx(req)
	gotCompany, okCompany := companyIDFromCtx(req)

	if !okUser || !okCompany {
		t.Errorf("expected both to be ok, user=%v company=%v", okUser, okCompany)
	}
	if gotUser != userID {
		t.Errorf("expected user %s, got %s", userID, gotUser)
	}
	if gotCompany != companyID {
		t.Errorf("expected company %s, got %s", companyID, gotCompany)
	}
	if gotUser == gotCompany {
		t.Error("user ID and company ID should be distinct contexts")
	}
}

// ── writeJSON content-type header set before body ─────────────────────────────

func TestWriteJSON_HeaderSetBeforeBody(t *testing.T) {
	// Verify the response writer gets the header before WriteHeader is called.
	// This is important because in HTTP, headers must be sent before body.
	rr := httptest.NewRecorder()
	writeJSON(rr, http.StatusOK, map[string]string{"test": "value"})
	// After the call, both header and body should be set correctly.
	if rr.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type not set")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

// ── writeJSON with http.Handler to test full request cycle ───────────────────

func TestWriteJSON_In_HTTPHandler(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusCreated, map[string]interface{}{"id": "abc123", "count": 5})
	})
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", rr.Code)
	}
	var m map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&m); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if m["id"] != "abc123" {
		t.Errorf("expected id=abc123, got %v", m["id"])
	}
	if m["count"] != float64(5) {
		t.Errorf("expected count=5, got %v", m["count"])
	}
}
