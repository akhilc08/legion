package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type contextKey string

const (
	ctxUserID    contextKey = "user_id"
	ctxCompanyID contextKey = "company_id"
)

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenStr := ""

		// Prefer Authorization header; fall back to ?token= for WebSocket upgrades.
		if h := r.Header.Get("Authorization"); h != "" {
			tokenStr = strings.TrimPrefix(h, "Bearer ")
		} else if q := r.URL.Query().Get("token"); q != "" {
			tokenStr = q
		}

		if tokenStr == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		claims := &jwt.RegisteredClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			return []byte(s.jwtSecret), nil
		})
		if err != nil || !token.Valid {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}

		userID, err := uuid.Parse(claims.Subject)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}

		ctx := context.WithValue(r.Context(), ctxUserID, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// companyAccessMiddleware reads the {companyID} chi URL param and validates access.
func (s *Server) companyAccessMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := userIDFromCtx(r)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}

		// Extract company ID from URL — chi stores it in the path context.
		// We parse it here for any route under /api/companies/{companyID}/.
		parts := strings.Split(r.URL.Path, "/")
		var companyIDStr string
		for i, p := range parts {
			if p == "companies" && i+1 < len(parts) {
				companyIDStr = parts[i+1]
				break
			}
		}

		companyID, err := uuid.Parse(companyIDStr)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid company id"})
			return
		}

		canAccess, err := s.db.UserCanAccessCompany(r.Context(), userID, companyID)
		if err != nil || !canAccess {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}

		ctx := context.WithValue(r.Context(), ctxCompanyID, companyID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func userIDFromCtx(r *http.Request) (uuid.UUID, bool) {
	v, ok := r.Context().Value(ctxUserID).(uuid.UUID)
	return v, ok
}

func companyIDFromCtx(r *http.Request) (uuid.UUID, bool) {
	v, ok := r.Context().Value(ctxCompanyID).(uuid.UUID)
	return v, ok
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint
}

func decodeJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}
