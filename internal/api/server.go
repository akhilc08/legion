package api

import (
	"context"
	"net/http"

	"conductor/internal/orchestrator"
	"conductor/internal/store"
	"conductor/internal/ws"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Server holds all API dependencies.
type Server struct {
	db        *store.DB
	hub       *ws.Hub
	orch      *orchestrator.Orchestrator
	jwtSecret string
	upgrader  websocket.Upgrader
	staticDir string // path to built React app
}

func NewServer(db *store.DB, hub *ws.Hub, orch *orchestrator.Orchestrator, jwtSecret, staticDir string) *Server {
	return &Server{
		db:        db,
		hub:       hub,
		orch:      orch,
		jwtSecret: jwtSecret,
		staticDir: staticDir,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)

	// Auth (public)
	r.Post("/api/auth/register", s.handleRegister)
	r.Post("/api/auth/login", s.handleLogin)

	// Runtime availability (public)
	r.Get("/api/runtimes", s.handleGetRuntimes)

	// Authenticated routes
	r.Group(func(r chi.Router) {
		r.Use(s.authMiddleware)

		// Companies
		r.Get("/api/companies", s.handleListCompanies)
		r.Post("/api/companies", s.handleCreateCompany)

		// Company-scoped routes
		r.Route("/api/companies/{companyID}", func(r chi.Router) {
			r.Use(s.chiCompanyMiddleware)

			r.Get("/", s.handleGetCompany)
			r.Patch("/", s.handleUpdateCompany)
			r.Delete("/", s.handleDeleteCompany)
			r.Post("/goal", s.handleSetGoal)

			// Agents
			r.Get("/agents", s.handleListAgents)
			r.Post("/agents", s.handleCreateAgent)
			r.Get("/agents/{agentID}", s.handleGetAgent)
			r.Delete("/agents/{agentID}", s.handleDeleteAgent)
			r.Post("/agents/{agentID}/spawn", s.handleSpawnAgent)
			r.Post("/agents/{agentID}/kill", s.handleKillAgent)
			r.Post("/agents/{agentID}/pause", s.handlePauseAgent)
			r.Post("/agents/{agentID}/resume", s.handleResumeAgent)
			r.Post("/agents/{agentID}/reassign", s.handleReassignAgent)
			r.Post("/agents/{agentID}/chat", s.handleAgentChat)
			r.Get("/agents/{agentID}/chat/history", s.handleChatHistory)

			// Issues
			r.Get("/issues", s.handleListIssues)
			r.Post("/issues", s.handleCreateIssue)
			r.Get("/issues/{issueID}", s.handleGetIssue)
			r.Patch("/issues/{issueID}", s.handleUpdateIssue)
			r.Post("/issues/{issueID}/dependencies", s.handleAddDependency)
			r.Delete("/issues/{issueID}/dependencies/{depID}", s.handleRemoveDependency)

			// Hiring
			r.Get("/hires", s.handleListPendingHires)
			r.Post("/hires/{hireID}/approve", s.handleApproveHire)
			r.Post("/hires/{hireID}/reject", s.handleRejectHire)

			// FS permissions
			r.Get("/fs/permissions", s.handleListFSPermissions)
			r.Post("/fs/permissions", s.handleGrantFSPermission)
			r.Delete("/fs/permissions/{permID}", s.handleRevokeFSPermission)

			// Audit log & notifications
			r.Get("/audit", s.handleListAuditLog)
			r.Get("/notifications", s.handleListNotifications)
			r.Post("/notifications/{notifID}/dismiss", s.handleDismissNotification)

			// WebSocket
			r.Get("/ws", s.handleWebSocket)
		})
	})

	// React SPA fallback
	r.Get("/*", s.handleSPA)

	return r
}

// chiCompanyMiddleware extracts companyID from chi URL params and validates access.
func (s *Server) chiCompanyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := userIDFromCtx(r)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}

		companyIDStr := chi.URLParam(r, "companyID")
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
