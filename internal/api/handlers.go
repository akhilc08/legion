package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"conductor/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// ── Auth ──────────────────────────────────────────────────────────────────────

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "hash failed"})
		return
	}
	user, err := s.db.CreateUser(r.Context(), body.Email, string(hash))
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "email already exists"})
		return
	}
	token, err := s.mintJWT(user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "mint token"})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{"token": token, "user": user})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	user, err := s.db.GetUserByEmail(r.Context(), body.Email)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(body.Password)); err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}
	token, err := s.mintJWT(user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "mint token"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"token": token, "user": user})
}

func (s *Server) mintJWT(userID uuid.UUID) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   userID.String(),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * 24 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.jwtSecret))
}

// ── Runtimes ──────────────────────────────────────────────────────────────────

func (s *Server) handleGetRuntimes(w http.ResponseWriter, r *http.Request) {
	av := s.orch.AvailableRuntimes()
	writeJSON(w, http.StatusOK, map[string]bool{
		"claude_code": av.ClaudeCode,
		"openclaw":    av.OpenClaw,
	})
}

// ── Companies ─────────────────────────────────────────────────────────────────

func (s *Server) handleListCompanies(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromCtx(r)
	companies, err := s.db.ListCompaniesForUser(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if companies == nil {
		companies = []store.Company{}
	}
	writeJSON(w, http.StatusOK, companies)
}

func (s *Server) handleCreateCompany(w http.ResponseWriter, r *http.Request) {
	userID, _ := userIDFromCtx(r)
	var body struct {
		Name string `json:"name"`
		Goal string `json:"goal"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	company, err := s.db.CreateCompany(r.Context(), body.Name, body.Goal)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	s.db.AddUserToCompany(r.Context(), userID, company.ID, "admin") //nolint
	writeJSON(w, http.StatusCreated, company)
}

func (s *Server) handleGetCompany(w http.ResponseWriter, r *http.Request) {
	companyID, _ := companyIDFromCtx(r)
	company, err := s.db.GetCompany(r.Context(), companyID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, company)
}

func (s *Server) handleUpdateCompany(w http.ResponseWriter, r *http.Request) {
	companyID, _ := companyIDFromCtx(r)
	var body struct {
		Goal string `json:"goal"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := s.db.UpdateCompanyGoal(r.Context(), companyID, body.Goal); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleDeleteCompany(w http.ResponseWriter, r *http.Request) {
	companyID, _ := companyIDFromCtx(r)
	if err := s.db.DeleteCompany(r.Context(), companyID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleSetGoal(w http.ResponseWriter, r *http.Request) {
	companyID, _ := companyIDFromCtx(r)
	var body struct {
		Goal string `json:"goal"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := s.db.UpdateCompanyGoal(r.Context(), companyID, body.Goal); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Deliver goal to CEO agent.
	ceo, err := s.db.GetCEO(r.Context(), companyID)
	if err == nil {
		issue := &store.Issue{
			CompanyID:   companyID,
			Title:       "Company Goal",
			Description: body.Goal,
			AssigneeID:  &ceo.ID,
		}
		s.db.CreateIssue(r.Context(), issue) //nolint
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ── Agents ────────────────────────────────────────────────────────────────────

func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	companyID, _ := companyIDFromCtx(r)
	agents, err := s.db.ListAgentsByCompany(r.Context(), companyID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if agents == nil {
		agents = []store.Agent{}
	}
	writeJSON(w, http.StatusOK, agents)
}

func (s *Server) handleCreateAgent(w http.ResponseWriter, r *http.Request) {
	companyID, _ := companyIDFromCtx(r)
	var body struct {
		Role          string             `json:"role"`
		Title         string             `json:"title"`
		SystemPrompt  string             `json:"system_prompt"`
		ManagerID     *uuid.UUID         `json:"manager_id"`
		Runtime       store.AgentRuntime `json:"runtime"`
		MonthlyBudget int                `json:"monthly_budget"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if body.Runtime == "" {
		body.Runtime = store.RuntimeClaudeCode
	}
	if body.MonthlyBudget == 0 {
		body.MonthlyBudget = 100000
	}
	agent := &store.Agent{
		CompanyID:     companyID,
		Role:          body.Role,
		Title:         body.Title,
		SystemPrompt:  body.SystemPrompt,
		ManagerID:     body.ManagerID,
		Runtime:       body.Runtime,
		MonthlyBudget: body.MonthlyBudget,
	}
	created, err := s.db.CreateAgent(r.Context(), agent)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) handleGetAgent(w http.ResponseWriter, r *http.Request) {
	agentID, err := uuid.Parse(chi.URLParam(r, "agentID"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid agent id"})
		return
	}
	agent, err := s.db.GetAgent(r.Context(), agentID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, agent)
}

func (s *Server) handleDeleteAgent(w http.ResponseWriter, r *http.Request) {
	agentID, err := uuid.Parse(chi.URLParam(r, "agentID"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid agent id"})
		return
	}
	s.orch.KillAgent(r.Context(), agentID) //nolint
	if err := s.db.DeleteAgent(r.Context(), agentID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleSpawnAgent(w http.ResponseWriter, r *http.Request) {
	agentID, err := uuid.Parse(chi.URLParam(r, "agentID"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid agent id"})
		return
	}
	agent, err := s.db.GetAgent(r.Context(), agentID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	if err := s.orch.SpawnAgent(r.Context(), agent); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "spawned"})
}

func (s *Server) handleKillAgent(w http.ResponseWriter, r *http.Request) {
	agentID, err := uuid.Parse(chi.URLParam(r, "agentID"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid agent id"})
		return
	}
	if err := s.orch.KillAgent(r.Context(), agentID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "killed"})
}

func (s *Server) handlePauseAgent(w http.ResponseWriter, r *http.Request) {
	agentID, err := uuid.Parse(chi.URLParam(r, "agentID"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid agent id"})
		return
	}
	if err := s.orch.PauseAgent(r.Context(), agentID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "paused"})
}

func (s *Server) handleResumeAgent(w http.ResponseWriter, r *http.Request) {
	agentID, err := uuid.Parse(chi.URLParam(r, "agentID"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid agent id"})
		return
	}
	if err := s.orch.ResumeAgent(r.Context(), agentID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "resumed"})
}

func (s *Server) handleReassignAgent(w http.ResponseWriter, r *http.Request) {
	agentID, err := uuid.Parse(chi.URLParam(r, "agentID"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid agent id"})
		return
	}
	var body struct {
		NewManagerID uuid.UUID `json:"new_manager_id"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := s.orch.ReassignAgent(r.Context(), agentID, body.NewManagerID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleAgentChat(w http.ResponseWriter, r *http.Request) {
	agentID, err := uuid.Parse(chi.URLParam(r, "agentID"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid agent id"})
		return
	}
	var body struct {
		Message string `json:"message"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	reply, err := s.orch.SendChatMessage(r.Context(), agentID, body.Message)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	// Ensure session exists and persist both turns.
	s.db.GetOrCreateChatSession(r.Context(), agentID) //nolint
	s.db.AppendChatMessage(r.Context(), agentID, "user", body.Message)      //nolint
	s.db.AppendChatMessage(r.Context(), agentID, "assistant", reply)        //nolint
	writeJSON(w, http.StatusOK, map[string]string{"reply": reply})
}

func (s *Server) handleChatHistory(w http.ResponseWriter, r *http.Request) {
	agentID, err := uuid.Parse(chi.URLParam(r, "agentID"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid agent id"})
		return
	}
	msgs, err := s.db.GetChatHistory(r.Context(), agentID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, msgs)
}

// ── Issues ────────────────────────────────────────────────────────────────────

func (s *Server) handleListIssues(w http.ResponseWriter, r *http.Request) {
	companyID, _ := companyIDFromCtx(r)
	issues, err := s.db.ListIssuesByCompany(r.Context(), companyID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if issues == nil {
		issues = []store.Issue{}
	}
	writeJSON(w, http.StatusOK, issues)
}

func (s *Server) handleCreateIssue(w http.ResponseWriter, r *http.Request) {
	companyID, _ := companyIDFromCtx(r)
	var body struct {
		Title       string     `json:"title"`
		Description string     `json:"description"`
		AssigneeID  *uuid.UUID `json:"assignee_id"`
		ParentID    *uuid.UUID `json:"parent_id"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	userID, _ := userIDFromCtx(r)
	_ = userID
	issue := &store.Issue{
		CompanyID:   companyID,
		Title:       body.Title,
		Description: body.Description,
		AssigneeID:  body.AssigneeID,
		ParentID:    body.ParentID,
	}
	created, err := s.db.CreateIssue(r.Context(), issue)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) handleGetIssue(w http.ResponseWriter, r *http.Request) {
	issueID, err := uuid.Parse(chi.URLParam(r, "issueID"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid issue id"})
		return
	}
	issue, err := s.db.GetIssue(r.Context(), issueID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, issue)
}

func (s *Server) handleUpdateIssue(w http.ResponseWriter, r *http.Request) {
	issueID, err := uuid.Parse(chi.URLParam(r, "issueID"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid issue id"})
		return
	}
	var body struct {
		AssigneeID *uuid.UUID        `json:"assignee_id"`
		Status     store.IssueStatus `json:"status"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if body.AssigneeID != nil {
		if err := s.db.UpdateIssueAssignee(r.Context(), issueID, *body.AssigneeID); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	}
	if body.Status != "" {
		if err := s.db.UpdateIssueStatus(r.Context(), issueID, body.Status); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleAddDependency(w http.ResponseWriter, r *http.Request) {
	issueID, err := uuid.Parse(chi.URLParam(r, "issueID"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid issue id"})
		return
	}
	var body struct {
		DependsOnID uuid.UUID `json:"depends_on_id"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := s.db.AddDependency(r.Context(), issueID, body.DependsOnID); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleRemoveDependency(w http.ResponseWriter, r *http.Request) {
	issueID, err := uuid.Parse(chi.URLParam(r, "issueID"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid issue id"})
		return
	}
	depID, err := uuid.Parse(chi.URLParam(r, "depID"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid dep id"})
		return
	}
	if err := s.db.RemoveDependency(r.Context(), issueID, depID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Hiring ────────────────────────────────────────────────────────────────────

func (s *Server) handleListPendingHires(w http.ResponseWriter, r *http.Request) {
	companyID, _ := companyIDFromCtx(r)
	hires, err := s.db.ListPendingHires(r.Context(), companyID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if hires == nil {
		hires = []store.PendingHire{}
	}
	writeJSON(w, http.StatusOK, hires)
}

func (s *Server) handleApproveHire(w http.ResponseWriter, r *http.Request) {
	hireID, err := uuid.Parse(chi.URLParam(r, "hireID"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid hire id"})
		return
	}
	if err := s.orch.ApproveHire(r.Context(), hireID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "approved"})
}

func (s *Server) handleRejectHire(w http.ResponseWriter, r *http.Request) {
	hireID, err := uuid.Parse(chi.URLParam(r, "hireID"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid hire id"})
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	decodeJSON(r, &body) //nolint
	if err := s.orch.RejectHire(r.Context(), hireID, body.Reason); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "rejected"})
}

// ── FS Permissions ────────────────────────────────────────────────────────────

func (s *Server) handleListFSPermissions(w http.ResponseWriter, r *http.Request) {
	companyID, _ := companyIDFromCtx(r)
	perms, err := s.db.ListFSPermissionsForCompany(r.Context(), companyID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if perms == nil {
		perms = []store.FSPermission{}
	}
	writeJSON(w, http.StatusOK, perms)
}

func (s *Server) handleGrantFSPermission(w http.ResponseWriter, r *http.Request) {
	var body struct {
		AgentID   uuid.UUID             `json:"agent_id"`
		Path      string                `json:"path"`
		Level     store.PermissionLevel `json:"permission_level"`
		GrantedBy *uuid.UUID            `json:"granted_by"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := s.db.GrantFSPermission(r.Context(), body.AgentID, body.Path, body.Level, body.GrantedBy); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleRevokeFSPermission(w http.ResponseWriter, r *http.Request) {
	permID, err := uuid.Parse(chi.URLParam(r, "permID"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid perm id"})
		return
	}
	if err := s.db.RevokeFSPermissionByID(r.Context(), permID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleFSBrowse(w http.ResponseWriter, r *http.Request) {
	companyID, _ := companyIDFromCtx(r)
	relPath := r.URL.Query().Get("path")
	if relPath == "" {
		relPath = "/"
	}
	fsRoot := filepath.Join("/conductor/companies", companyID.String(), "fs")
	target := filepath.Join(fsRoot, filepath.Clean("/"+relPath))
	if !strings.HasPrefix(target, fsRoot) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid path"})
		return
	}
	entries, err := os.ReadDir(target)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, http.StatusOK, []interface{}{})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	type FSEntry struct {
		Name  string `json:"name"`
		IsDir bool   `json:"is_dir"`
		Size  int64  `json:"size"`
	}
	result := make([]FSEntry, 0, len(entries))
	for _, e := range entries {
		info, _ := e.Info()
		size := int64(0)
		if info != nil {
			size = info.Size()
		}
		result = append(result, FSEntry{Name: e.Name(), IsDir: e.IsDir(), Size: size})
	}
	writeJSON(w, http.StatusOK, result)
}

// ── Audit & Notifications ─────────────────────────────────────────────────────

func (s *Server) handleListAuditLog(w http.ResponseWriter, r *http.Request) {
	companyID, _ := companyIDFromCtx(r)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	logs, err := s.db.ListAuditLog(r.Context(), companyID, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, logs)
}

func (s *Server) handleListNotifications(w http.ResponseWriter, r *http.Request) {
	companyID, _ := companyIDFromCtx(r)
	ns, err := s.db.ListActiveNotifications(r.Context(), companyID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, ns)
}

func (s *Server) handleDismissNotification(w http.ResponseWriter, r *http.Request) {
	notifID, err := uuid.Parse(chi.URLParam(r, "notifID"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := s.db.DismissNotification(r.Context(), notifID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ── WebSocket ─────────────────────────────────────────────────────────────────

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	companyID, _ := companyIDFromCtx(r)

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	<-s.hub.Register(conn, companyID)
}

// ── SPA ───────────────────────────────────────────────────────────────────────

func (s *Server) handleSPA(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, s.staticDir+"/index.html")
}
