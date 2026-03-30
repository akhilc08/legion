package stdout

import (
	"encoding/json"
	"strings"
	"testing"

	"conductor/internal/agent"
	"conductor/internal/store"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// IsControlLine
// ---------------------------------------------------------------------------

func TestIsControlLine(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		// Exact prefix matches (5)
		{"exact HIRE", "CONDUCTOR_HIRE", true},
		{"exact ESCALATE", "CONDUCTOR_ESCALATE", true},
		{"exact DONE", "CONDUCTOR_DONE", true},
		{"exact BLOCKED", "CONDUCTOR_BLOCKED", true},
		{"exact HEARTBEAT", "CONDUCTOR_HEARTBEAT", true},

		// Prefix followed by space+braces (5)
		{"HIRE space braces", "CONDUCTOR_HIRE {}", true},
		{"ESCALATE space braces", "CONDUCTOR_ESCALATE {}", true},
		{"DONE space braces", "CONDUCTOR_DONE {}", true},
		{"BLOCKED space braces", "CONDUCTOR_BLOCKED {}", true},
		{"HEARTBEAT space braces", "CONDUCTOR_HEARTBEAT {}", true},

		// Prefix followed by JSON (5)
		{"HIRE with JSON", `CONDUCTOR_HIRE {"role_title":"coder"}`, true},
		{"ESCALATE with JSON", `CONDUCTOR_ESCALATE {"issue_id":"00000000-0000-0000-0000-000000000001"}`, true},
		{"DONE with JSON", `CONDUCTOR_DONE {"output_path":"/tmp/out"}`, true},
		{"BLOCKED with JSON", `CONDUCTOR_BLOCKED {"waiting_on_issue_id":"00000000-0000-0000-0000-000000000002"}`, true},
		{"HEARTBEAT with JSON", `CONDUCTOR_HEARTBEAT {"ts":1234567890}`, true},

		// Lowercase versions → false (5)
		{"lowercase hire", "conductor_hire", false},
		{"lowercase escalate", "conductor_escalate", false},
		{"lowercase done", "conductor_done", false},
		{"lowercase blocked", "conductor_blocked", false},
		{"lowercase heartbeat", "conductor_heartbeat", false},

		// Partial prefix → false (5)
		{"partial HIRE", "CONDUCTOR_HIR", false},
		{"partial ESCALATE", "CONDUCTOR_ESCALA", false},
		{"partial DONE", "CONDUCTOR_DON", false},
		{"partial BLOCKED", "CONDUCTOR_BLOCKE", false},
		{"partial HEARTBEAT", "CONDUCTOR_HEARTBEA", false},

		// Prefix with extra characters at start → false (5)
		{"leading space HIRE", " CONDUCTOR_HIRE", false},
		{"leading X ESCALATE", "XCONDUCTOR_ESCALATE", false},
		{"leading newline DONE", "\nCONDUCTOR_DONE", false},
		{"leading tab BLOCKED", "\tCONDUCTOR_BLOCKED", false},
		{"leading dash HEARTBEAT", "-CONDUCTOR_HEARTBEAT", false},

		// Empty string → false
		{"empty string", "", false},

		// Random strings → false (4)
		{"random hello", "hello world", false},
		{"CONDUCTOR alone", "CONDUCTOR", false},
		{"CONDUCTOR_", "CONDUCTOR_", false},
		{"similar prefix", "CONDUCTOR_HIRES", true}, // HasPrefix matches CONDUCTOR_HIRE

		// Whitespace variations → false
		{"only whitespace", "   ", false},
		{"tab only", "\t", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsControlLine(tt.input)
			if got != tt.want {
				t.Errorf("IsControlLine(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ParseLine
// ---------------------------------------------------------------------------

func TestParseLine(t *testing.T) {
	// helpers
	rawJSON := func(s string) json.RawMessage { return json.RawMessage(s) }
	emptyPayload := rawJSON("{}")

	tests := []struct {
		name        string
		input       string
		wantPrefix  string
		wantPayload json.RawMessage
		wantErr     bool
	}{
		// Each prefix with valid JSON payload (5)
		{
			"HIRE valid JSON",
			`CONDUCTOR_HIRE {"role_title":"coder","system_prompt":"you are a coder"}`,
			agent.ControlHire,
			rawJSON(`{"role_title":"coder","system_prompt":"you are a coder"}`),
			false,
		},
		{
			"ESCALATE valid JSON",
			`CONDUCTOR_ESCALATE {"issue_id":"00000000-0000-0000-0000-000000000001","reason":"stuck"}`,
			agent.ControlEscalate,
			rawJSON(`{"issue_id":"00000000-0000-0000-0000-000000000001","reason":"stuck"}`),
			false,
		},
		{
			"DONE valid JSON",
			`CONDUCTOR_DONE {"output_path":"/out","tokens_used":100}`,
			agent.ControlDone,
			rawJSON(`{"output_path":"/out","tokens_used":100}`),
			false,
		},
		{
			"BLOCKED valid JSON",
			`CONDUCTOR_BLOCKED {"waiting_on_issue_id":"00000000-0000-0000-0000-000000000002"}`,
			agent.ControlBlocked,
			rawJSON(`{"waiting_on_issue_id":"00000000-0000-0000-0000-000000000002"}`),
			false,
		},
		{
			"HEARTBEAT valid JSON",
			`CONDUCTOR_HEARTBEAT {"ts":999}`,
			agent.ControlHeartbeat,
			rawJSON(`{"ts":999}`),
			false,
		},

		// Each prefix with no payload → defaults to "{}" (5)
		{"HIRE no payload", "CONDUCTOR_HIRE", agent.ControlHire, emptyPayload, false},
		{"ESCALATE no payload", "CONDUCTOR_ESCALATE", agent.ControlEscalate, emptyPayload, false},
		{"DONE no payload", "CONDUCTOR_DONE", agent.ControlDone, emptyPayload, false},
		{"BLOCKED no payload", "CONDUCTOR_BLOCKED", agent.ControlBlocked, emptyPayload, false},
		{"HEARTBEAT no payload", "CONDUCTOR_HEARTBEAT", agent.ControlHeartbeat, emptyPayload, false},

		// Each prefix with whitespace then JSON (5)
		{
			"HIRE whitespace then JSON",
			"CONDUCTOR_HIRE   {\"role_title\":\"x\"}",
			agent.ControlHire,
			rawJSON(`{"role_title":"x"}`),
			false,
		},
		{
			"ESCALATE whitespace then JSON",
			"CONDUCTOR_ESCALATE  {}",
			agent.ControlEscalate,
			rawJSON(`{}`),
			false,
		},
		{
			"DONE whitespace then JSON",
			"CONDUCTOR_DONE \t{}",
			agent.ControlDone,
			rawJSON(`{}`),
			false,
		},
		{
			"BLOCKED whitespace then JSON",
			"CONDUCTOR_BLOCKED  {}",
			agent.ControlBlocked,
			rawJSON(`{}`),
			false,
		},
		{
			"HEARTBEAT whitespace then JSON",
			"CONDUCTOR_HEARTBEAT   {}",
			agent.ControlHeartbeat,
			rawJSON(`{}`),
			false,
		},

		// Each prefix with invalid JSON → error (5)
		{"HIRE invalid JSON", "CONDUCTOR_HIRE {bad}", "", nil, true},
		{"ESCALATE invalid JSON", "CONDUCTOR_ESCALATE not-json", "", nil, true},
		{"DONE invalid JSON", "CONDUCTOR_DONE {\"x\":}", "", nil, true},
		{"BLOCKED invalid JSON", "CONDUCTOR_BLOCKED [[[", "", nil, true},
		{"HEARTBEAT invalid JSON", "CONDUCTOR_HEARTBEAT }", "", nil, true},

		// Non-control line → error
		{"non-control line", "hello world", "", nil, true},
		{"random text", "some random text", "", nil, true},

		// Empty string → error
		{"empty string", "", "", nil, true},

		// Prefix only (no space after) → succeeds with "{}" default
		{"HIRE prefix only no space", "CONDUCTOR_HIRE", agent.ControlHire, emptyPayload, false},

		// Prefix with trailing whitespace only → succeeds with "{}" default
		{"HIRE trailing whitespace", "CONDUCTOR_HIRE   ", agent.ControlHire, emptyPayload, false},
		{"DONE trailing whitespace", "CONDUCTOR_DONE\t", agent.ControlDone, emptyPayload, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseLine(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseLine(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseLine(%q) unexpected error: %v", tt.input, err)
				return
			}
			if got.Prefix != tt.wantPrefix {
				t.Errorf("ParseLine(%q).Prefix = %q, want %q", tt.input, got.Prefix, tt.wantPrefix)
			}
			if string(got.Payload) != string(tt.wantPayload) {
				t.Errorf("ParseLine(%q).Payload = %s, want %s", tt.input, got.Payload, tt.wantPayload)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// DecodeHire
// ---------------------------------------------------------------------------

func TestDecodeHire(t *testing.T) {
	validUUID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	tests := []struct {
		name             string
		input            string
		wantErr          bool
		wantRoleTitle    string
		wantSystemPrompt string
		wantBudget       int
	}{
		// Valid full payload
		{
			name:             "valid full payload",
			input:            `{"role_title":"coder","system_prompt":"you are a coder","budget_allocation":10000,"reporting_to":"00000000-0000-0000-0000-000000000001","runtime":"claude_code"}`,
			wantRoleTitle:    "coder",
			wantSystemPrompt: "you are a coder",
			wantBudget:       10000,
		},
		// Missing role_title → error
		{
			name:    "missing role_title",
			input:   `{"system_prompt":"you are a coder","budget_allocation":5000}`,
			wantErr: true,
		},
		// Missing system_prompt → error
		{
			name:    "missing system_prompt",
			input:   `{"role_title":"coder","budget_allocation":5000}`,
			wantErr: true,
		},
		// Zero budget → defaults to 50000
		{
			name:             "zero budget defaults to 50000",
			input:            `{"role_title":"coder","system_prompt":"do stuff","budget_allocation":0}`,
			wantRoleTitle:    "coder",
			wantSystemPrompt: "do stuff",
			wantBudget:       50000,
		},
		// Negative budget → defaults to 50000
		{
			name:             "negative budget defaults to 50000",
			input:            `{"role_title":"coder","system_prompt":"do stuff","budget_allocation":-100}`,
			wantRoleTitle:    "coder",
			wantSystemPrompt: "do stuff",
			wantBudget:       50000,
		},
		// Budget set to 100 → keeps 100
		{
			name:             "budget 100 kept",
			input:            `{"role_title":"reviewer","system_prompt":"review code","budget_allocation":100}`,
			wantRoleTitle:    "reviewer",
			wantSystemPrompt: "review code",
			wantBudget:       100,
		},
		// Valid with all fields including optional initial_task
		{
			name:             "valid all fields with initial_task",
			input:            `{"role_title":"tester","system_prompt":"write tests","budget_allocation":2000,"initial_task":"test everything","reporting_to":"00000000-0000-0000-0000-000000000001","runtime":"claude_code"}`,
			wantRoleTitle:    "tester",
			wantSystemPrompt: "write tests",
			wantBudget:       2000,
		},
		// Invalid JSON → error
		{
			name:    "invalid JSON",
			input:   `{bad json}`,
			wantErr: true,
		},
		// Empty JSON "{}" → error (missing required fields)
		{
			name:    "empty JSON object",
			input:   `{}`,
			wantErr: true,
		},
		// Budget omitted (not present) → defaults to 50000
		{
			name:             "budget omitted defaults to 50000",
			input:            `{"role_title":"writer","system_prompt":"write docs"}`,
			wantRoleTitle:    "writer",
			wantSystemPrompt: "write docs",
			wantBudget:       50000,
		},
		// Only role_title missing system_prompt
		{
			name:    "only role_title no system_prompt",
			input:   `{"role_title":"coder"}`,
			wantErr: true,
		},
		// Only system_prompt missing role_title
		{
			name:    "only system_prompt no role_title",
			input:   `{"system_prompt":"do stuff"}`,
			wantErr: true,
		},
	}

	_ = validUUID // used above in table literal

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeHire(json.RawMessage(tt.input))
			if tt.wantErr {
				if err == nil {
					t.Errorf("DecodeHire(%s) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("DecodeHire(%s) unexpected error: %v", tt.input, err)
				return
			}
			if got.RoleTitle != tt.wantRoleTitle {
				t.Errorf("RoleTitle = %q, want %q", got.RoleTitle, tt.wantRoleTitle)
			}
			if got.SystemPrompt != tt.wantSystemPrompt {
				t.Errorf("SystemPrompt = %q, want %q", got.SystemPrompt, tt.wantSystemPrompt)
			}
			if got.BudgetAllocation != tt.wantBudget {
				t.Errorf("BudgetAllocation = %d, want %d", got.BudgetAllocation, tt.wantBudget)
			}
		})
	}
}

// TestDecodeHireRuntimeField verifies the Runtime field is decoded properly.
func TestDecodeHireRuntimeField(t *testing.T) {
	raw := json.RawMessage(`{"role_title":"coder","system_prompt":"code","runtime":"claude_code","budget_allocation":1}`)
	got, err := DecodeHire(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Runtime != store.RuntimeClaudeCode {
		t.Errorf("Runtime = %q, want %q", got.Runtime, store.RuntimeClaudeCode)
	}
}

// TestDecodeHireReportingTo verifies the ReportingTo UUID is decoded.
func TestDecodeHireReportingTo(t *testing.T) {
	id := uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	raw := json.RawMessage(`{"role_title":"coder","system_prompt":"code","reporting_to":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee","budget_allocation":1}`)
	got, err := DecodeHire(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ReportingTo != id {
		t.Errorf("ReportingTo = %v, want %v", got.ReportingTo, id)
	}
}

// ---------------------------------------------------------------------------
// DecodeEscalate
// ---------------------------------------------------------------------------

func TestDecodeEscalate(t *testing.T) {
	issueID := uuid.MustParse("00000000-0000-0000-0000-000000000010")

	tests := []struct {
		name       string
		input      string
		wantErr    bool
		wantIssue  uuid.UUID
		wantReason string
	}{
		{
			name:       "valid payload",
			input:      `{"issue_id":"00000000-0000-0000-0000-000000000010","reason":"need help"}`,
			wantIssue:  issueID,
			wantReason: "need help",
		},
		{
			name:  "empty object",
			input: `{}`,
			// No required fields — decodes fine with zero values
			wantIssue:  uuid.Nil,
			wantReason: "",
		},
		{
			name:    "invalid JSON",
			input:   `{bad}`,
			wantErr: true,
		},
		{
			name:       "only reason field",
			input:      `{"reason":"blocked on api"}`,
			wantIssue:  uuid.Nil,
			wantReason: "blocked on api",
		},
		{
			name:      "only issue_id field",
			input:     `{"issue_id":"00000000-0000-0000-0000-000000000010"}`,
			wantIssue: issueID,
		},
		{
			name:    "invalid uuid value",
			input:   `{"issue_id":"not-a-uuid","reason":"x"}`,
			wantErr: true,
		},
		{
			name:    "array instead of object",
			input:   `[]`,
			wantErr: true,
		},
		{
			name:    "null input",
			input:   `null`,
			// json.Unmarshal of null into struct is a no-op (no error)
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeEscalate(json.RawMessage(tt.input))
			if tt.wantErr {
				if err == nil {
					t.Errorf("DecodeEscalate(%s) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("DecodeEscalate(%s) unexpected error: %v", tt.input, err)
				return
			}
			if got == nil {
				t.Fatalf("got nil result")
			}
			if got.IssueID != tt.wantIssue {
				t.Errorf("IssueID = %v, want %v", got.IssueID, tt.wantIssue)
			}
			if got.Reason != tt.wantReason {
				t.Errorf("Reason = %q, want %q", got.Reason, tt.wantReason)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// DecodeDone
// ---------------------------------------------------------------------------

func TestDecodeDone(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		wantErr         bool
		wantOutputPath  string
		wantTokensUsed  int
	}{
		{
			name:           "valid payload",
			input:          `{"output_path":"/tmp/result.txt","tokens_used":1234}`,
			wantOutputPath: "/tmp/result.txt",
			wantTokensUsed: 1234,
		},
		{
			name:           "empty object zero values",
			input:          `{}`,
			wantOutputPath: "",
			wantTokensUsed: 0,
		},
		{
			name:    "invalid JSON",
			input:   `{bad}`,
			wantErr: true,
		},
		{
			name:           "only output_path",
			input:          `{"output_path":"/out"}`,
			wantOutputPath: "/out",
			wantTokensUsed: 0,
		},
		{
			name:           "only tokens_used",
			input:          `{"tokens_used":500}`,
			wantOutputPath: "",
			wantTokensUsed: 500,
		},
		{
			name:    "truncated JSON",
			input:   `{"output_path":`,
			wantErr: true,
		},
		{
			name:    "number instead of object",
			input:   `42`,
			wantErr: true,
		},
		{
			name:           "null input",
			input:          `null`,
			wantOutputPath: "",
			wantTokensUsed: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeDone(json.RawMessage(tt.input))
			if tt.wantErr {
				if err == nil {
					t.Errorf("DecodeDone(%s) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("DecodeDone(%s) unexpected error: %v", tt.input, err)
				return
			}
			if got == nil {
				t.Fatalf("got nil result")
			}
			if got.OutputPath != tt.wantOutputPath {
				t.Errorf("OutputPath = %q, want %q", got.OutputPath, tt.wantOutputPath)
			}
			if got.TokensUsed != tt.wantTokensUsed {
				t.Errorf("TokensUsed = %d, want %d", got.TokensUsed, tt.wantTokensUsed)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// DecodeBlocked
// ---------------------------------------------------------------------------

func TestDecodeBlocked(t *testing.T) {
	waitID := uuid.MustParse("00000000-0000-0000-0000-000000000099")

	tests := []struct {
		name        string
		input       string
		wantErr     bool
		wantWaitOn  uuid.UUID
	}{
		{
			name:       "valid payload",
			input:      `{"waiting_on_issue_id":"00000000-0000-0000-0000-000000000099"}`,
			wantWaitOn: waitID,
		},
		{
			name:       "empty object zero values",
			input:      `{}`,
			wantWaitOn: uuid.Nil,
		},
		{
			name:    "invalid JSON",
			input:   `{bad}`,
			wantErr: true,
		},
		{
			name:    "invalid uuid",
			input:   `{"waiting_on_issue_id":"not-uuid"}`,
			wantErr: true,
		},
		{
			name:    "array input",
			input:   `[]`,
			wantErr: true,
		},
		{
			name:       "null input",
			input:      `null`,
			wantWaitOn: uuid.Nil,
		},
		{
			name:    "string input",
			input:   `"hello"`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeBlocked(json.RawMessage(tt.input))
			if tt.wantErr {
				if err == nil {
					t.Errorf("DecodeBlocked(%s) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("DecodeBlocked(%s) unexpected error: %v", tt.input, err)
				return
			}
			if got == nil {
				t.Fatalf("got nil result")
			}
			if got.WaitingOnIssueID != tt.wantWaitOn {
				t.Errorf("WaitingOnIssueID = %v, want %v", got.WaitingOnIssueID, tt.wantWaitOn)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Round-trip: ParseLine → Decode*
// ---------------------------------------------------------------------------

func TestParseLineDecodeRoundTrip(t *testing.T) {
	hireID := uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-000000000001")

	t.Run("HIRE round trip", func(t *testing.T) {
		line := `CONDUCTOR_HIRE {"role_title":"backend","system_prompt":"write APIs","budget_allocation":200,"reporting_to":"aaaaaaaa-bbbb-cccc-dddd-000000000001","runtime":"claude_code"}`
		pc, err := ParseLine(line)
		if err != nil {
			t.Fatalf("ParseLine: %v", err)
		}
		h, err := DecodeHire(pc.Payload)
		if err != nil {
			t.Fatalf("DecodeHire: %v", err)
		}
		if h.RoleTitle != "backend" {
			t.Errorf("RoleTitle = %q", h.RoleTitle)
		}
		if h.BudgetAllocation != 200 {
			t.Errorf("BudgetAllocation = %d", h.BudgetAllocation)
		}
		if h.ReportingTo != hireID {
			t.Errorf("ReportingTo = %v", h.ReportingTo)
		}
	})

	t.Run("ESCALATE round trip", func(t *testing.T) {
		line := `CONDUCTOR_ESCALATE {"issue_id":"00000000-0000-0000-0000-000000000001","reason":"need review"}`
		pc, err := ParseLine(line)
		if err != nil {
			t.Fatalf("ParseLine: %v", err)
		}
		e, err := DecodeEscalate(pc.Payload)
		if err != nil {
			t.Fatalf("DecodeEscalate: %v", err)
		}
		if e.Reason != "need review" {
			t.Errorf("Reason = %q", e.Reason)
		}
	})

	t.Run("DONE round trip", func(t *testing.T) {
		line := `CONDUCTOR_DONE {"output_path":"/artifacts/result","tokens_used":42}`
		pc, err := ParseLine(line)
		if err != nil {
			t.Fatalf("ParseLine: %v", err)
		}
		d, err := DecodeDone(pc.Payload)
		if err != nil {
			t.Fatalf("DecodeDone: %v", err)
		}
		if d.OutputPath != "/artifacts/result" {
			t.Errorf("OutputPath = %q", d.OutputPath)
		}
		if d.TokensUsed != 42 {
			t.Errorf("TokensUsed = %d", d.TokensUsed)
		}
	})

	t.Run("BLOCKED round trip", func(t *testing.T) {
		line := `CONDUCTOR_BLOCKED {"waiting_on_issue_id":"00000000-0000-0000-0000-000000000099"}`
		pc, err := ParseLine(line)
		if err != nil {
			t.Fatalf("ParseLine: %v", err)
		}
		b, err := DecodeBlocked(pc.Payload)
		if err != nil {
			t.Fatalf("DecodeBlocked: %v", err)
		}
		expected := uuid.MustParse("00000000-0000-0000-0000-000000000099")
		if b.WaitingOnIssueID != expected {
			t.Errorf("WaitingOnIssueID = %v", b.WaitingOnIssueID)
		}
	})
}

// ---------------------------------------------------------------------------
// Verify ParsedControl struct fields are exported
// ---------------------------------------------------------------------------

func TestParsedControlFields(t *testing.T) {
	pc := &ParsedControl{
		Prefix:  "CONDUCTOR_HIRE",
		Payload: json.RawMessage(`{}`),
	}
	if pc.Prefix == "" {
		t.Error("Prefix should not be empty")
	}
	if pc.Payload == nil {
		t.Error("Payload should not be nil")
	}
}

// ---------------------------------------------------------------------------
// Verify error messages contain useful context
// ---------------------------------------------------------------------------

func TestParseLineErrorMessages(t *testing.T) {
	_, err := ParseLine("CONDUCTOR_HIRE {bad}")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "CONDUCTOR_HIRE") {
		t.Errorf("error should mention the prefix, got: %v", err)
	}
}

func TestDecodeHireErrorMessages(t *testing.T) {
	_, err := DecodeHire(json.RawMessage(`{"system_prompt":"x"}`))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "role_title") {
		t.Errorf("error should mention role_title, got: %v", err)
	}

	_, err = DecodeHire(json.RawMessage(`{"role_title":"coder"}`))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "system_prompt") {
		t.Errorf("error should mention system_prompt, got: %v", err)
	}
}
