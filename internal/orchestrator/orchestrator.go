// Package orchestrator is the central coordinator: it manages agent lifecycle,
// task assignment, failure handling, and escalation bubbling.
package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"conductor/internal/agent"
	"conductor/internal/heartbeat"
	"conductor/internal/store"
	"conductor/internal/ws"
	"github.com/google/uuid"
)

// Runtimes tracks which agent CLIs are available on this machine.
type Runtimes struct {
	ClaudeCode bool
	OpenClaw   bool
}

// Orchestrator is the central engine. One instance per server process.
type Orchestrator struct {
	db      *store.DB
	hub     *ws.Hub
	watcher *heartbeat.Watcher
	fsRoot  string // base directory for company file systems

	mu       sync.RWMutex
	// Live runtime instances keyed by agent ID
	runtimes map[uuid.UUID]agent.AgentRuntime
	available Runtimes

	// Runtime availability re-check ticker
	runtimeCheckStop chan struct{}
}

// New creates and initializes an Orchestrator.
func New(db *store.DB, hub *ws.Hub, fsRoot string) *Orchestrator {
	o := &Orchestrator{
		db:               db,
		hub:              hub,
		fsRoot:           fsRoot,
		runtimes:         make(map[uuid.UUID]agent.AgentRuntime),
		runtimeCheckStop: make(chan struct{}),
	}

	o.watcher = heartbeat.NewWatcher(db,
		func(ctx context.Context, agentID, companyID uuid.UUID) {
			o.handleAgentFailed(ctx, agentID, companyID)
		},
		func(ctx context.Context, agentID, companyID uuid.UUID) {
			o.handleAgentDegraded(ctx, agentID, companyID)
		},
	)

	o.refreshRuntimes()
	return o
}

// Start begins background goroutines: runtime availability checker and boot reconciler.
func (o *Orchestrator) Start(ctx context.Context) error {
	// Reconcile Postgres agent state with running processes on boot.
	if err := o.reconcileOnBoot(ctx); err != nil {
		log.Printf("orchestrator: reconcile on boot: %v", err)
	}

	// Periodically re-check runtime availability.
	go func() {
		t := time.NewTicker(5 * time.Minute)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				o.refreshRuntimes()
			case <-o.runtimeCheckStop:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

// Stop shuts down all background goroutines gracefully.
func (o *Orchestrator) Stop() {
	close(o.runtimeCheckStop)
}

// AvailableRuntimes returns which runtimes are currently detected.
func (o *Orchestrator) AvailableRuntimes() Runtimes {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.available
}

func (o *Orchestrator) refreshRuntimes() {
	cc := commandExists("claude")
	oc := commandExists("openclaw")

	o.mu.Lock()
	prev := o.available
	o.available = Runtimes{ClaudeCode: cc, OpenClaw: oc}
	changed := prev != o.available
	o.mu.Unlock()

	if changed {
		log.Printf("orchestrator: runtimes updated — claude_code:%v openclaw:%v", cc, oc)
		// Notify all clients of the change.
		o.hub.BroadcastAll(ws.Event{
			Type: ws.EventRuntimeStatus,
			Payload: map[string]bool{
				"claude_code": cc,
				"openclaw":    oc,
			},
		})

		// If a runtime disappeared, mark affected agents DEGRADED.
		if !cc && prev.ClaudeCode {
			go o.markAgentsDegraded(context.Background(), store.RuntimeClaudeCode)
		}
		if !oc && prev.OpenClaw {
			go o.markAgentsDegraded(context.Background(), store.RuntimeOpenClaw)
		}
	}
}

func (o *Orchestrator) markAgentsDegraded(ctx context.Context, runtime store.AgentRuntime) {
	// List all companies and find agents using this runtime that are active.
	companies, err := o.db.ListCompanies(ctx)
	if err != nil {
		return
	}
	for _, c := range companies {
		agents, err := o.db.ListAgentsByCompany(ctx, c.ID)
		if err != nil {
			continue
		}
		for _, a := range agents {
			if a.Runtime == runtime && (a.Status == store.StatusIdle || a.Status == store.StatusWorking) {
				o.db.UpdateAgentStatus(ctx, a.ID, store.StatusDegraded) //nolint
				o.hub.Broadcast(ws.Event{
					Type:      ws.EventAgentStatus,
					CompanyID: c.ID,
					Payload:   map[string]interface{}{"agent_id": a.ID, "status": store.StatusDegraded},
				})
			}
		}
	}
}

// SpawnAgent starts a new agent subprocess and registers it with the heartbeat watcher.
func (o *Orchestrator) SpawnAgent(ctx context.Context, a *store.Agent) error {
	workDir := filepath.Join(o.fsRoot, a.CompanyID.String(), "fs")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return fmt.Errorf("create workdir: %w", err)
	}

	config := agent.AgentConfig{
		AgentID:      a.ID,
		CompanyID:    a.CompanyID,
		SystemPrompt: a.SystemPrompt,
		WorkDir:      workDir,
		EnvVars: map[string]string{
			"CONDUCTOR_COMPANY_ID": a.CompanyID.String(),
		},
	}

	rt, err := o.createRuntime(a.Runtime, a.ID)
	if err != nil {
		return fmt.Errorf("create runtime: %w", err)
	}

	if err := rt.Spawn(ctx, config); err != nil {
		return fmt.Errorf("spawn agent %s: %w", a.ID, err)
	}

	pid := rt.PID()
	if err := o.db.UpdateAgentPID(ctx, a.ID, &pid); err != nil {
		log.Printf("orchestrator: update pid: %v", err)
	}
	if err := o.db.UpdateAgentStatus(ctx, a.ID, store.StatusIdle); err != nil {
		log.Printf("orchestrator: update status idle: %v", err)
	}
	if err := o.db.UpsertHeartbeat(ctx, a.ID); err != nil {
		log.Printf("orchestrator: upsert heartbeat: %v", err)
	}

	o.mu.Lock()
	o.runtimes[a.ID] = rt
	o.mu.Unlock()

	o.watcher.Watch(ctx, a.ID, a.CompanyID)
	o.hub.Broadcast(ws.Event{
		Type:      ws.EventAgentStatus,
		CompanyID: a.CompanyID,
		Payload:   map[string]interface{}{"agent_id": a.ID, "status": store.StatusIdle, "pid": pid},
	})

	// Start issue assignment loop for this agent.
	go o.assignLoop(ctx, a.ID, a.CompanyID)

	return nil
}

// KillAgent terminates an agent and cleans up.
func (o *Orchestrator) KillAgent(ctx context.Context, agentID uuid.UUID) error {
	o.mu.Lock()
	rt, ok := o.runtimes[agentID]
	if ok {
		delete(o.runtimes, agentID)
	}
	o.mu.Unlock()

	o.watcher.Unwatch(agentID)

	if ok {
		if err := rt.Kill(ctx); err != nil {
			log.Printf("orchestrator: kill agent %s: %v", agentID, err)
		}
	}

	return o.db.UpdateAgentStatus(ctx, agentID, store.StatusFailed)
}

// RespawnAgent kills and restarts an agent, restoring its last task context.
func (o *Orchestrator) RespawnAgent(ctx context.Context, agentID uuid.UUID) error {
	a, err := o.db.GetAgent(ctx, agentID)
	if err != nil {
		return fmt.Errorf("get agent for respawn: %w", err)
	}

	o.KillAgent(ctx, agentID) //nolint — ignore kill errors, process may already be gone

	return o.SpawnAgent(ctx, a)
}

// PauseAgent pauses an agent for human chat.
func (o *Orchestrator) PauseAgent(ctx context.Context, agentID uuid.UUID) error {
	o.mu.RLock()
	rt, ok := o.runtimes[agentID]
	o.mu.RUnlock()
	if !ok {
		return fmt.Errorf("agent not running")
	}

	if err := rt.Pause(ctx); err != nil {
		return err
	}
	if err := o.db.UpdateAgentStatus(ctx, agentID, store.StatusPaused); err != nil {
		return err
	}

	a, _ := o.db.GetAgent(ctx, agentID)
	if a != nil {
		o.hub.Broadcast(ws.Event{
			Type:      ws.EventAgentStatus,
			CompanyID: a.CompanyID,
			Payload:   map[string]interface{}{"agent_id": agentID, "status": store.StatusPaused},
		})
	}
	return nil
}

// ResumeAgent resumes a paused agent.
func (o *Orchestrator) ResumeAgent(ctx context.Context, agentID uuid.UUID) error {
	o.mu.RLock()
	rt, ok := o.runtimes[agentID]
	o.mu.RUnlock()
	if !ok {
		return fmt.Errorf("agent not running")
	}

	if err := rt.Resume(ctx); err != nil {
		return err
	}
	if err := o.db.UpdateAgentStatus(ctx, agentID, store.StatusWorking); err != nil {
		return err
	}

	a, _ := o.db.GetAgent(ctx, agentID)
	if a != nil {
		o.hub.Broadcast(ws.Event{
			Type:      ws.EventAgentStatus,
			CompanyID: a.CompanyID,
			Payload:   map[string]interface{}{"agent_id": agentID, "status": store.StatusWorking},
		})
	}
	return nil
}

// SendChatMessage sends a message to a paused agent and returns the response.
func (o *Orchestrator) SendChatMessage(ctx context.Context, agentID uuid.UUID, message string) (string, error) {
	o.mu.RLock()
	rt, ok := o.runtimes[agentID]
	o.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("agent not running")
	}

	reply, err := rt.SendChat(ctx, message)
	if err != nil {
		return "", err
	}

	a, _ := o.db.GetAgent(ctx, agentID)
	if a != nil {
		o.hub.Broadcast(ws.Event{
			Type:      ws.EventChatMessage,
			CompanyID: a.CompanyID,
			Payload: map[string]interface{}{
				"agent_id": agentID,
				"role":     "assistant",
				"content":  reply,
			},
		})
	}
	return reply, nil
}

// assignLoop periodically checks for ready issues and assigns them to an idle agent.
func (o *Orchestrator) assignLoop(ctx context.Context, agentID, companyID uuid.UUID) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a, err := o.db.GetAgent(ctx, agentID)
			if err != nil || a.Status != store.StatusIdle {
				continue
			}

			issues, err := o.db.ListReadyIssues(ctx, companyID)
			if err != nil {
				log.Printf("assignLoop: list ready: %v", err)
				continue
			}

			for _, issue := range issues {
				if issue.AssigneeID != nil && *issue.AssigneeID != agentID {
					continue
				}

				checked, err := o.db.CheckoutIssue(ctx, issue.ID, agentID)
				if err != nil || !checked {
					continue
				}

				if err := o.db.UpdateAgentStatus(ctx, agentID, store.StatusWorking); err != nil {
					log.Printf("assignLoop: update status: %v", err)
				}

				o.mu.RLock()
				rt, ok := o.runtimes[agentID]
				o.mu.RUnlock()
				if !ok {
					break
				}

				if err := rt.SendTask(ctx, issue); err != nil {
					log.Printf("assignLoop: send task: %v", err)
				}

				o.db.Log(ctx, companyID, &agentID, "ISSUE_ASSIGNED", map[string]interface{}{
					"issue_id": issue.ID, "agent_id": agentID,
				}) //nolint

				o.hub.Broadcast(ws.Event{
					Type:      ws.EventIssueUpdate,
					CompanyID: companyID,
					Payload: map[string]interface{}{
						"issue_id": issue.ID, "status": store.IssueInProgress, "assignee_id": agentID,
					},
				})
				break
			}
		}
	}
}

// HandleControlMessage dispatches a parsed control message from an agent.
func (o *Orchestrator) HandleControlMessage(ctx context.Context, agentID, companyID uuid.UUID, prefix string, raw json.RawMessage) {
	switch prefix {
	case agent.ControlHeartbeat:
		if err := o.watcher.RecordHeartbeat(ctx, agentID); err != nil {
			log.Printf("orchestrator: heartbeat record: %v", err)
		}
		o.db.UpdateAgentStatus(ctx, agentID, store.StatusWorking) //nolint

	case agent.ControlDone:
		var d agent.DonePayload
		if err := json.Unmarshal(raw, &d); err == nil {
			o.handleDone(ctx, agentID, companyID, d)
		}

	case agent.ControlEscalate:
		var e agent.EscalatePayload
		if err := json.Unmarshal(raw, &e); err == nil {
			o.handleEscalate(ctx, agentID, companyID, e)
		}

	case agent.ControlBlocked:
		var b agent.BlockedPayload
		if err := json.Unmarshal(raw, &b); err == nil {
			o.handleBlocked(ctx, agentID, companyID, b)
		}

	case agent.ControlHire:
		var h agent.HirePayload
		if err := json.Unmarshal(raw, &h); err == nil {
			o.handleHireRequest(ctx, agentID, companyID, h)
		}
	}
}

func (o *Orchestrator) handleDone(ctx context.Context, agentID, companyID uuid.UUID, d agent.DonePayload) {
	// Find the in-progress issue for this agent and mark done.
	issues, err := o.db.ListIssuesByCompany(ctx, companyID)
	if err != nil {
		return
	}
	for _, issue := range issues {
		if issue.AssigneeID != nil && *issue.AssigneeID == agentID && issue.Status == store.IssueInProgress {
			o.db.UpdateIssueOutput(ctx, issue.ID, d.OutputPath) //nolint
			o.db.UpdateAgentStatus(ctx, agentID, store.StatusIdle) //nolint
			if d.TokensUsed > 0 {
				o.db.AddTokenSpend(ctx, agentID, d.TokensUsed, false) //nolint
			}
			o.db.Log(ctx, companyID, &agentID, "ISSUE_COMPLETED", map[string]interface{}{
				"issue_id": issue.ID, "output_path": d.OutputPath,
			}) //nolint
			o.hub.Broadcast(ws.Event{
				Type:      ws.EventIssueUpdate,
				CompanyID: companyID,
				Payload:   map[string]interface{}{"issue_id": issue.ID, "status": store.IssueDone},
			})
			break
		}
	}
}

func (o *Orchestrator) handleBlocked(ctx context.Context, agentID, companyID uuid.UUID, b agent.BlockedPayload) {
	issues, err := o.db.ListIssuesByCompany(ctx, companyID)
	if err != nil {
		return
	}
	for _, issue := range issues {
		if issue.AssigneeID != nil && *issue.AssigneeID == agentID && issue.Status == store.IssueInProgress {
			o.db.UpdateIssueStatus(ctx, issue.ID, store.IssueBlocked) //nolint
			o.db.AddDependency(ctx, issue.ID, b.WaitingOnIssueID) //nolint
			o.db.UpdateAgentStatus(ctx, agentID, store.StatusBlocked) //nolint
			o.hub.Broadcast(ws.Event{
				Type:      ws.EventIssueUpdate,
				CompanyID: companyID,
				Payload:   map[string]interface{}{"issue_id": issue.ID, "status": store.IssueBlocked},
			})
			break
		}
	}
}

func (o *Orchestrator) handleEscalate(ctx context.Context, agentID, companyID uuid.UUID, e agent.EscalatePayload) {
	issue, err := o.db.GetIssue(ctx, e.IssueID)
	if err != nil {
		return
	}

	o.db.IncrementAttemptCount(ctx, e.IssueID, e.Reason) //nolint

	a, err := o.db.GetAgent(ctx, agentID)
	if err != nil || a.ManagerID == nil {
		// CEO-level escalation → notify human
		o.notifyHumanEscalation(ctx, companyID, issue, e.Reason)
		return
	}

	// Find or create escalation record
	chainEntry := store.EscalationChainEntry{
		AgentID:     agentID,
		Reason:      e.Reason,
		AttemptedAt: time.Now(),
	}
	_ = chainEntry

	o.db.Log(ctx, companyID, &agentID, "ESCALATION_BUBBLED", map[string]interface{}{
		"issue_id": e.IssueID, "from_agent": agentID, "to_manager": a.ManagerID, "reason": e.Reason,
	}) //nolint

	// Reassign issue to manager
	o.db.UpdateIssueAssignee(ctx, e.IssueID, *a.ManagerID) //nolint
	o.db.UpdateIssueStatus(ctx, e.IssueID, store.IssuePending) //nolint

	o.hub.Broadcast(ws.Event{
		Type:      ws.EventEscalation,
		CompanyID: companyID,
		Payload: map[string]interface{}{
			"issue_id":    e.IssueID,
			"from_agent":  agentID,
			"to_agent":    a.ManagerID,
			"reason":      e.Reason,
		},
	})
}

func (o *Orchestrator) notifyHumanEscalation(ctx context.Context, companyID uuid.UUID, issue *store.Issue, reason string) {
	n := &store.Notification{
		CompanyID: companyID,
		Type:      "escalation_human",
		Payload: map[string]interface{}{
			"issue_id":    issue.ID,
			"issue_title": issue.Title,
			"reason":      reason,
		},
	}
	n, err := o.db.CreateNotification(ctx, n)
	if err != nil {
		log.Printf("orchestrator: create escalation notification: %v", err)
		return
	}

	o.hub.Broadcast(ws.Event{
		Type:      ws.EventNotification,
		CompanyID: companyID,
		Payload: map[string]interface{}{
			"notification_id": n.ID,
			"type":            "escalation_human",
			"issue_title":     issue.Title,
			"reason":          reason,
		},
	})

	o.db.Log(ctx, companyID, nil, "CEO_HUMAN_ALERT", map[string]interface{}{
		"issue_id": issue.ID, "reason": reason,
	}) //nolint
}

func (o *Orchestrator) handleHireRequest(ctx context.Context, agentID, companyID uuid.UUID, h agent.HirePayload) {
	// Validate: reporting_to must be within hiring agent's subtree
	subtree, err := o.db.GetSubtreeAgentIDs(ctx, agentID)
	if err != nil {
		log.Printf("orchestrator: hire subtree check: %v", err)
		return
	}

	inSubtree := false
	for _, id := range subtree {
		if id == h.ReportingTo {
			inSubtree = true
			break
		}
	}
	if !inSubtree && h.ReportingTo != agentID {
		// Reject: cannot hire outside own subtree
		o.sendRejection(ctx, agentID, "hire rejected: reporting_to is outside your subtree")
		return
	}

	// Validate budget
	requester, err := o.db.GetAgent(ctx, agentID)
	if err != nil {
		return
	}
	remaining := requester.MonthlyBudget - requester.TokenSpend
	if h.BudgetAllocation > remaining {
		o.sendRejection(ctx, agentID, fmt.Sprintf("hire rejected: insufficient budget (have %d, need %d)", remaining, h.BudgetAllocation))
		return
	}

	// Validate runtime
	o.mu.RLock()
	av := o.available
	o.mu.RUnlock()
	if (h.Runtime == store.RuntimeClaudeCode && !av.ClaudeCode) ||
		(h.Runtime == store.RuntimeOpenClaw && !av.OpenClaw) {
		o.sendRejection(ctx, agentID, fmt.Sprintf("hire rejected: runtime %s not available", h.Runtime))
		return
	}

	initialTask := h.InitialTask
	pending := &store.PendingHire{
		CompanyID:          companyID,
		RequestedByAgentID: agentID,
		RoleTitle:          h.RoleTitle,
		ReportingToAgentID: h.ReportingTo,
		SystemPrompt:       h.SystemPrompt,
		Runtime:            h.Runtime,
		BudgetAllocation:   h.BudgetAllocation,
		InitialTask:        &initialTask,
	}

	created, err := o.db.CreatePendingHire(ctx, pending)
	if err != nil {
		log.Printf("orchestrator: create pending hire: %v", err)
		return
	}

	o.db.Log(ctx, companyID, &agentID, "HIRE_REQUESTED", map[string]interface{}{
		"hire_id":    created.ID,
		"role_title": h.RoleTitle,
	}) //nolint

	o.hub.Broadcast(ws.Event{
		Type:      ws.EventHirePending,
		CompanyID: companyID,
		Payload: map[string]interface{}{
			"hire_id":    created.ID,
			"role_title": h.RoleTitle,
			"requested_by": agentID,
		},
	})
}

func (o *Orchestrator) sendRejection(ctx context.Context, agentID uuid.UUID, reason string) {
	o.mu.RLock()
	rt, ok := o.runtimes[agentID]
	o.mu.RUnlock()
	if !ok {
		return
	}
	rt.SendTask(ctx, store.Issue{Title: "REJECTION", Description: reason}) //nolint
}

// ApproveHire spawns a new agent for an approved pending hire.
func (o *Orchestrator) ApproveHire(ctx context.Context, hireID uuid.UUID) error {
	hire, err := o.db.GetPendingHire(ctx, hireID)
	if err != nil {
		return fmt.Errorf("get pending hire: %w", err)
	}
	if hire.Status != store.HirePending {
		return fmt.Errorf("hire %s is not pending", hireID)
	}

	// Deduct budget from requesting manager
	requester, err := o.db.GetAgent(ctx, hire.RequestedByAgentID)
	if err != nil {
		return err
	}
	if err := o.db.AddTokenSpend(ctx, requester.ID, hire.BudgetAllocation, false); err != nil {
		return fmt.Errorf("deduct budget: %w", err)
	}

	// Create the new agent
	newAgent := &store.Agent{
		CompanyID:     hire.CompanyID,
		Role:          hire.RoleTitle,
		Title:         hire.RoleTitle,
		SystemPrompt:  hire.SystemPrompt,
		ManagerID:     &hire.ReportingToAgentID,
		Runtime:       hire.Runtime,
		MonthlyBudget: hire.BudgetAllocation,
	}
	created, err := o.db.CreateAgent(ctx, newAgent)
	if err != nil {
		return fmt.Errorf("create agent: %w", err)
	}

	// Grant FS permissions cascaded from manager
	if err := o.db.CascadePermissionsFromManager(ctx, created.ID, hire.ReportingToAgentID); err != nil {
		log.Printf("orchestrator: cascade permissions: %v", err)
	}

	// Spawn the agent
	if err := o.SpawnAgent(ctx, created); err != nil {
		return fmt.Errorf("spawn new agent: %w", err)
	}

	// Create initial task issue if provided
	if hire.InitialTask != nil && *hire.InitialTask != "" {
		issue := &store.Issue{
			CompanyID:   hire.CompanyID,
			Title:       fmt.Sprintf("Initial task for %s", hire.RoleTitle),
			Description: *hire.InitialTask,
			AssigneeID:  &created.ID,
			CreatedBy:   &hire.RequestedByAgentID,
		}
		o.db.CreateIssue(ctx, issue) //nolint
	}

	o.db.UpdateHireStatus(ctx, hireID, store.HireApproved) //nolint
	o.db.Log(ctx, hire.CompanyID, nil, "HIRE_APPROVED", map[string]interface{}{
		"hire_id": hireID, "new_agent_id": created.ID, "role_title": hire.RoleTitle,
	}) //nolint

	o.hub.Broadcast(ws.Event{
		Type:      ws.EventAgentStatus,
		CompanyID: hire.CompanyID,
		Payload: map[string]interface{}{
			"agent_id": created.ID, "status": store.StatusIdle, "event": "hired",
		},
	})

	return nil
}

// RejectHire rejects a pending hire and notifies the requesting agent.
func (o *Orchestrator) RejectHire(ctx context.Context, hireID uuid.UUID, reason string) error {
	hire, err := o.db.GetPendingHire(ctx, hireID)
	if err != nil {
		return err
	}
	o.db.UpdateHireStatus(ctx, hireID, store.HireRejected) //nolint
	o.db.Log(ctx, hire.CompanyID, nil, "HIRE_REJECTED", map[string]interface{}{
		"hire_id": hireID, "reason": reason,
	}) //nolint
	o.sendRejection(ctx, hire.RequestedByAgentID, fmt.Sprintf("hire rejected by human: %s", reason))
	return nil
}

// ReassignAgent changes an agent's manager (org chart drag-drop).
func (o *Orchestrator) ReassignAgent(ctx context.Context, agentID, newManagerID uuid.UUID) error {
	if err := o.db.UpdateAgentManager(ctx, agentID, newManagerID); err != nil {
		return fmt.Errorf("update manager: %w", err)
	}

	// Cascade FS permissions from new manager
	if err := o.db.CascadePermissionsFromManager(ctx, agentID, newManagerID); err != nil {
		log.Printf("orchestrator: cascade after reassign: %v", err)
	}

	a, _ := o.db.GetAgent(ctx, agentID)
	if a != nil {
		o.db.Log(ctx, a.CompanyID, nil, "HIERARCHY_CHANGE", map[string]interface{}{
			"agent_id": agentID, "new_manager_id": newManagerID,
		}) //nolint
		o.hub.Broadcast(ws.Event{
			Type:      ws.EventAgentStatus,
			CompanyID: a.CompanyID,
			Payload: map[string]interface{}{
				"agent_id": agentID, "new_manager_id": newManagerID, "event": "reassigned",
			},
		})
	}
	return nil
}

func (o *Orchestrator) handleAgentFailed(ctx context.Context, agentID, companyID uuid.UUID) {
	log.Printf("orchestrator: agent %s failed — respawning", agentID)

	// Release any in-progress issues back to pending.
	issues, err := o.db.ListIssuesByCompany(ctx, companyID)
	if err == nil {
		for _, issue := range issues {
			if issue.AssigneeID != nil && *issue.AssigneeID == agentID && issue.Status == store.IssueInProgress {
				o.db.UpdateIssueStatus(ctx, issue.ID, store.IssuePending) //nolint
			}
		}
	}

	if err := o.RespawnAgent(ctx, agentID); err != nil {
		log.Printf("orchestrator: respawn %s: %v", agentID, err)
	}

	o.hub.Broadcast(ws.Event{
		Type:      ws.EventAgentStatus,
		CompanyID: companyID,
		Payload:   map[string]interface{}{"agent_id": agentID, "event": "respawned"},
	})
}

func (o *Orchestrator) handleAgentDegraded(ctx context.Context, agentID, companyID uuid.UUID) {
	o.hub.Broadcast(ws.Event{
		Type:      ws.EventAgentStatus,
		CompanyID: companyID,
		Payload:   map[string]interface{}{"agent_id": agentID, "status": store.StatusDegraded},
	})

	n := &store.Notification{
		CompanyID: companyID,
		Type:      "agent_degraded",
		Payload:   map[string]interface{}{"agent_id": agentID},
	}
	o.db.CreateNotification(ctx, n) //nolint
}

// reconcileOnBoot reconciles agents that were running before a server restart.
func (o *Orchestrator) reconcileOnBoot(ctx context.Context) error {
	companies, err := o.db.ListCompanies(ctx)
	if err != nil {
		return err
	}

	for _, c := range companies {
		agents, err := o.db.ListAgentsByCompany(ctx, c.ID)
		if err != nil {
			log.Printf("reconcile: list agents for %s: %v", c.ID, err)
			continue
		}
		for _, a := range agents {
			if a.Status == store.StatusIdle || a.Status == store.StatusWorking ||
				a.Status == store.StatusPaused || a.Status == store.StatusBlocked {
				log.Printf("reconcile: respawning agent %s (%s)", a.ID, a.Role)
				go func(ag store.Agent) {
					if err := o.SpawnAgent(ctx, &ag); err != nil {
						log.Printf("reconcile: spawn %s: %v", ag.ID, err)
					}
				}(a)
			}
		}
	}
	return nil
}

func (o *Orchestrator) createRuntime(runtime store.AgentRuntime, agentID uuid.UUID) (agent.AgentRuntime, error) {
	outputHandler := func(line string) {
		// Broadcast task output to UI clients
		ctx := context.Background()
		a, err := o.db.GetAgent(ctx, agentID)
		if err != nil {
			return
		}
		o.hub.Broadcast(ws.Event{
			Type:      ws.EventAgentLog,
			CompanyID: a.CompanyID,
			Payload:   map[string]interface{}{"agent_id": agentID, "line": line},
		})
	}

	controlHandler := func(prefix string, raw json.RawMessage) {
		ctx := context.Background()
		a, err := o.db.GetAgent(ctx, agentID)
		if err != nil {
			return
		}
		o.HandleControlMessage(ctx, agentID, a.CompanyID, prefix, raw)
	}

	switch runtime {
	case store.RuntimeClaudeCode:
		return agent.NewClaudeCodeRuntime(outputHandler, controlHandler), nil
	case store.RuntimeOpenClaw:
		return agent.NewOpenClawRuntime(outputHandler, controlHandler), nil
	default:
		return nil, fmt.Errorf("unknown runtime: %s", runtime)
	}
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
