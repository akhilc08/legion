package agent

import (
	"encoding/json"
	"testing"

	"conductor/internal/store"
	"github.com/google/uuid"
)

// ── controlPrefixes ───────────────────────────────────────────────────────────

func TestControlPrefixes_ContainsAllFive(t *testing.T) {
	expected := []string{
		ControlHire,
		ControlEscalate,
		ControlDone,
		ControlBlocked,
		ControlHeartbeat,
	}
	if len(controlPrefixes) != len(expected) {
		t.Fatalf("expected %d control prefixes, got %d", len(expected), len(controlPrefixes))
	}
	set := make(map[string]bool, len(controlPrefixes))
	for _, p := range controlPrefixes {
		set[p] = true
	}
	for _, e := range expected {
		if !set[e] {
			t.Errorf("controlPrefixes missing %q", e)
		}
	}
}

func TestControlPrefixes_ConstantValues(t *testing.T) {
	if ControlHire != "CONDUCTOR_HIRE" {
		t.Errorf("ControlHire = %q, want CONDUCTOR_HIRE", ControlHire)
	}
	if ControlEscalate != "CONDUCTOR_ESCALATE" {
		t.Errorf("ControlEscalate = %q, want CONDUCTOR_ESCALATE", ControlEscalate)
	}
	if ControlDone != "CONDUCTOR_DONE" {
		t.Errorf("ControlDone = %q, want CONDUCTOR_DONE", ControlDone)
	}
	if ControlBlocked != "CONDUCTOR_BLOCKED" {
		t.Errorf("ControlBlocked = %q, want CONDUCTOR_BLOCKED", ControlBlocked)
	}
	if ControlHeartbeat != "CONDUCTOR_HEARTBEAT" {
		t.Errorf("ControlHeartbeat = %q, want CONDUCTOR_HEARTBEAT", ControlHeartbeat)
	}
}

// ── Payload types ─────────────────────────────────────────────────────────────

func TestHirePayload_RoundTrip(t *testing.T) {
	id := uuid.New()
	original := HirePayload{
		RoleTitle:        "engineer",
		ReportingTo:      id,
		SystemPrompt:     "You are an engineer",
		Runtime:          store.RuntimeClaudeCode,
		BudgetAllocation: 5000,
		InitialTask:      "build a thing",
	}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded HirePayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.RoleTitle != original.RoleTitle {
		t.Errorf("RoleTitle: got %q, want %q", decoded.RoleTitle, original.RoleTitle)
	}
	if decoded.ReportingTo != original.ReportingTo {
		t.Errorf("ReportingTo: got %v, want %v", decoded.ReportingTo, original.ReportingTo)
	}
	if decoded.BudgetAllocation != original.BudgetAllocation {
		t.Errorf("BudgetAllocation: got %d, want %d", decoded.BudgetAllocation, original.BudgetAllocation)
	}
	if decoded.InitialTask != original.InitialTask {
		t.Errorf("InitialTask: got %q, want %q", decoded.InitialTask, original.InitialTask)
	}
}

func TestHirePayload_InitialTaskOmittedWhenEmpty(t *testing.T) {
	p := HirePayload{RoleTitle: "dev"}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// omitempty means "initial_task" key should not appear
	m := make(map[string]interface{})
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := m["initial_task"]; ok {
		t.Error("expected initial_task to be omitted when empty")
	}
}

func TestEscalatePayload_RoundTrip(t *testing.T) {
	issueID := uuid.New()
	original := EscalatePayload{
		IssueID: issueID,
		Reason:  "blocked on external dep",
	}
	data, _ := json.Marshal(original)
	var decoded EscalatePayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.IssueID != original.IssueID {
		t.Errorf("IssueID: got %v, want %v", decoded.IssueID, original.IssueID)
	}
	if decoded.Reason != original.Reason {
		t.Errorf("Reason: got %q, want %q", decoded.Reason, original.Reason)
	}
}

func TestDonePayload_RoundTrip(t *testing.T) {
	original := DonePayload{
		OutputPath: "/tmp/output.txt",
		TokensUsed: 1234,
	}
	data, _ := json.Marshal(original)
	var decoded DonePayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.OutputPath != original.OutputPath {
		t.Errorf("OutputPath: got %q, want %q", decoded.OutputPath, original.OutputPath)
	}
	if decoded.TokensUsed != original.TokensUsed {
		t.Errorf("TokensUsed: got %d, want %d", decoded.TokensUsed, original.TokensUsed)
	}
}

func TestBlockedPayload_RoundTrip(t *testing.T) {
	id := uuid.New()
	original := BlockedPayload{WaitingOnIssueID: id}
	data, _ := json.Marshal(original)
	var decoded BlockedPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.WaitingOnIssueID != original.WaitingOnIssueID {
		t.Errorf("WaitingOnIssueID: got %v, want %v", decoded.WaitingOnIssueID, original.WaitingOnIssueID)
	}
}

func TestEscalatePayload_JSONFieldNames(t *testing.T) {
	p := EscalatePayload{Reason: "test"}
	data, _ := json.Marshal(p)
	m := make(map[string]interface{})
	json.Unmarshal(data, &m) //nolint
	if _, ok := m["issue_id"]; !ok {
		t.Error("expected 'issue_id' field")
	}
	if _, ok := m["reason"]; !ok {
		t.Error("expected 'reason' field")
	}
}

func TestDonePayload_JSONFieldNames(t *testing.T) {
	p := DonePayload{OutputPath: "/x", TokensUsed: 1}
	data, _ := json.Marshal(p)
	m := make(map[string]interface{})
	json.Unmarshal(data, &m) //nolint
	if _, ok := m["output_path"]; !ok {
		t.Error("expected 'output_path' field")
	}
	if _, ok := m["tokens_used"]; !ok {
		t.Error("expected 'tokens_used' field")
	}
}
