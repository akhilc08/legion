package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"testing"
	"time"
)

// ── NewClaudeCodeRuntime ──────────────────────────────────────────────────────

func TestNewClaudeCodeRuntime_NilHandlers(t *testing.T) {
	r := NewClaudeCodeRuntime(nil, nil)
	if r == nil {
		t.Fatal("expected non-nil runtime")
	}
}

func TestNewClaudeCodeRuntime_WithHandlers(t *testing.T) {
	called := false
	r := NewClaudeCodeRuntime(
		func(line string) { called = true },
		func(prefix string, payload json.RawMessage) { called = true },
	)
	if r.OutputHandler == nil {
		t.Error("expected OutputHandler to be set")
	}
	if r.ControlHandler == nil {
		t.Error("expected ControlHandler to be set")
	}
	_ = called
}

// ── PID before spawn ──────────────────────────────────────────────────────────

func TestClaudeCodePID_BeforeSpawn(t *testing.T) {
	r := NewClaudeCodeRuntime(nil, nil)
	if r.PID() != 0 {
		t.Errorf("expected PID 0 before spawn, got %d", r.PID())
	}
}

// ── TokensUsed before spawn ───────────────────────────────────────────────────

func TestClaudeCodeTokensUsed_BeforeSpawn(t *testing.T) {
	r := NewClaudeCodeRuntime(nil, nil)
	if r.TokensUsed() != 0 {
		t.Errorf("expected 0 tokens before spawn, got %d", r.TokensUsed())
	}
}

// ── Kill with no process ──────────────────────────────────────────────────────

func TestClaudeCodeKill_NoProcess(t *testing.T) {
	r := NewClaudeCodeRuntime(nil, nil)
	err := r.Kill(context.Background())
	if err != nil {
		t.Errorf("expected nil error killing idle runtime, got %v", err)
	}
}

// ── Heartbeat with no process ─────────────────────────────────────────────────

func TestClaudeCodeHeartbeat_NoProcess(t *testing.T) {
	r := NewClaudeCodeRuntime(nil, nil)
	err := r.Heartbeat(context.Background())
	if err != nil {
		t.Errorf("expected nil for idle heartbeat (idle is ok), got %v", err)
	}
}

// ── Pause and Resume ──────────────────────────────────────────────────────────

func TestClaudeCodePause_NoProcess(t *testing.T) {
	r := NewClaudeCodeRuntime(nil, nil)
	if err := r.Pause(context.Background()); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestClaudeCodeResume_NoProcess(t *testing.T) {
	r := NewClaudeCodeRuntime(nil, nil)
	if err := r.Resume(context.Background()); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestClaudeCodePauseResume_Sequence(t *testing.T) {
	r := NewClaudeCodeRuntime(nil, nil)
	if err := r.Pause(context.Background()); err != nil {
		t.Errorf("Pause: %v", err)
	}
	if err := r.Resume(context.Background()); err != nil {
		t.Errorf("Resume: %v", err)
	}
}

// ── SendChat not supported ────────────────────────────────────────────────────

func TestClaudeCodeSendChat_ReturnsError(t *testing.T) {
	r := NewClaudeCodeRuntime(nil, nil)
	_, err := r.SendChat(context.Background(), "hello")
	if err == nil {
		t.Error("expected error from SendChat, got nil")
	}
}

// ── Spawn stores config ───────────────────────────────────────────────────────

func TestClaudeCodeSpawn_StoresConfig(t *testing.T) {
	r := NewClaudeCodeRuntime(nil, nil)
	config := AgentConfig{WorkDir: "/tmp/test-workdir"}
	err := r.Spawn(context.Background(), config)
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	r.mu.Lock()
	stored := r.config
	r.mu.Unlock()
	if stored.WorkDir != "/tmp/test-workdir" {
		t.Errorf("expected WorkDir=/tmp/test-workdir, got %q", stored.WorkDir)
	}
}

func TestClaudeCodeSpawn_DoesNotStartProcess(t *testing.T) {
	r := NewClaudeCodeRuntime(nil, nil)
	err := r.Spawn(context.Background(), AgentConfig{WorkDir: "/tmp"})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	if r.PID() != 0 {
		t.Error("expected PID=0 after Spawn (no process started)")
	}
}

// ── tryParseTokens ────────────────────────────────────────────────────────────

func TestTryParseTokens_ValidUsageMessage(t *testing.T) {
	r := NewClaudeCodeRuntime(nil, nil)
	line := `{"type":"usage","usage":{"input_tokens":100,"output_tokens":50}}`
	r.tryParseTokens(line)
	if r.TokensUsed() != 150 {
		t.Errorf("expected 150 tokens, got %d", r.TokensUsed())
	}
}

func TestTryParseTokens_NonUsageType(t *testing.T) {
	r := NewClaudeCodeRuntime(nil, nil)
	line := `{"type":"assistant","content":"hello"}`
	r.tryParseTokens(line)
	if r.TokensUsed() != 0 {
		t.Errorf("expected 0 tokens for non-usage message, got %d", r.TokensUsed())
	}
}

func TestTryParseTokens_InvalidJSON(t *testing.T) {
	r := NewClaudeCodeRuntime(nil, nil)
	r.tryParseTokens("not json at all")
	if r.TokensUsed() != 0 {
		t.Errorf("expected 0 for invalid JSON, got %d", r.TokensUsed())
	}
}

func TestTryParseTokens_EmptyString(t *testing.T) {
	r := NewClaudeCodeRuntime(nil, nil)
	r.tryParseTokens("")
	if r.TokensUsed() != 0 {
		t.Errorf("expected 0 for empty string, got %d", r.TokensUsed())
	}
}

func TestTryParseTokens_Accumulates(t *testing.T) {
	r := NewClaudeCodeRuntime(nil, nil)
	r.tryParseTokens(`{"type":"usage","usage":{"input_tokens":10,"output_tokens":5}}`)
	r.tryParseTokens(`{"type":"usage","usage":{"input_tokens":20,"output_tokens":10}}`)
	r.tryParseTokens(`{"type":"usage","usage":{"input_tokens":5,"output_tokens":5}}`)
	if r.TokensUsed() != 55 {
		t.Errorf("expected 55 accumulated tokens, got %d", r.TokensUsed())
	}
}

func TestTryParseTokens_ZeroTokens(t *testing.T) {
	r := NewClaudeCodeRuntime(nil, nil)
	r.tryParseTokens(`{"type":"usage","usage":{"input_tokens":0,"output_tokens":0}}`)
	// A usage message with zero tokens is still valid; tokens stay at 0
	if r.TokensUsed() != 0 {
		t.Errorf("expected 0, got %d", r.TokensUsed())
	}
}

func TestTryParseTokens_OnlyInputTokens(t *testing.T) {
	r := NewClaudeCodeRuntime(nil, nil)
	r.tryParseTokens(`{"type":"usage","usage":{"input_tokens":42,"output_tokens":0}}`)
	if r.TokensUsed() != 42 {
		t.Errorf("expected 42, got %d", r.TokensUsed())
	}
}

// ── handleControl ─────────────────────────────────────────────────────────────

func TestHandleControl_NilHandler(t *testing.T) {
	r := NewClaudeCodeRuntime(nil, nil)
	// Should not panic when ControlHandler is nil
	r.handleControl("CONDUCTOR_HEARTBEAT {}")
}

func TestHandleControl_KnownPrefix_Heartbeat(t *testing.T) {
	var gotPrefix string
	var gotPayload json.RawMessage
	done := make(chan struct{}, 1)
	r := NewClaudeCodeRuntime(nil, func(prefix string, payload json.RawMessage) {
		gotPrefix = prefix
		gotPayload = payload
		done <- struct{}{}
	})
	r.handleControl("CONDUCTOR_HEARTBEAT {}")
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("control handler not called")
	}
	if gotPrefix != ControlHeartbeat {
		t.Errorf("expected prefix %q, got %q", ControlHeartbeat, gotPrefix)
	}
	if string(gotPayload) != "{}" {
		t.Errorf("expected payload '{}', got %q", gotPayload)
	}
}

func TestHandleControl_KnownPrefix_Hire(t *testing.T) {
	done := make(chan string, 1)
	r := NewClaudeCodeRuntime(nil, func(prefix string, _ json.RawMessage) {
		done <- prefix
	})
	r.handleControl(`CONDUCTOR_HIRE {"role_title":"eng"}`)
	select {
	case p := <-done:
		if p != ControlHire {
			t.Errorf("expected %q, got %q", ControlHire, p)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("control handler not called")
	}
}

func TestHandleControl_KnownPrefix_Escalate(t *testing.T) {
	done := make(chan string, 1)
	r := NewClaudeCodeRuntime(nil, func(prefix string, _ json.RawMessage) {
		done <- prefix
	})
	r.handleControl(`CONDUCTOR_ESCALATE {"reason":"stuck"}`)
	select {
	case p := <-done:
		if p != ControlEscalate {
			t.Errorf("expected %q, got %q", ControlEscalate, p)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("control handler not called")
	}
}

func TestHandleControl_KnownPrefix_Done(t *testing.T) {
	done := make(chan string, 1)
	r := NewClaudeCodeRuntime(nil, func(prefix string, _ json.RawMessage) {
		done <- prefix
	})
	r.handleControl(`CONDUCTOR_DONE {"output_path":"/tmp/out"}`)
	select {
	case p := <-done:
		if p != ControlDone {
			t.Errorf("expected %q, got %q", ControlDone, p)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("control handler not called")
	}
}

func TestHandleControl_KnownPrefix_Blocked(t *testing.T) {
	done := make(chan string, 1)
	r := NewClaudeCodeRuntime(nil, func(prefix string, _ json.RawMessage) {
		done <- prefix
	})
	r.handleControl(`CONDUCTOR_BLOCKED {}`)
	select {
	case p := <-done:
		if p != ControlBlocked {
			t.Errorf("expected %q, got %q", ControlBlocked, p)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("control handler not called")
	}
}

func TestHandleControl_NoPayload_UsesEmptyJSON(t *testing.T) {
	var gotPayload json.RawMessage
	done := make(chan struct{}, 1)
	r := NewClaudeCodeRuntime(nil, func(_ string, payload json.RawMessage) {
		gotPayload = payload
		done <- struct{}{}
	})
	// Control message with no payload (bare prefix)
	r.handleControl("CONDUCTOR_HEARTBEAT")
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("control handler not called")
	}
	if string(gotPayload) != "{}" {
		t.Errorf("expected '{}' for empty payload, got %q", gotPayload)
	}
}

func TestHandleControl_UnknownLine_NotCalled(t *testing.T) {
	called := false
	r := NewClaudeCodeRuntime(nil, func(_ string, _ json.RawMessage) {
		called = true
	})
	r.handleControl("some random output line")
	time.Sleep(20 * time.Millisecond)
	if called {
		t.Error("control handler should not be called for unknown prefix")
	}
}

// ── readOutput via pipe ───────────────────────────────────────────────────────

func TestReadOutput_ControlLineNotPassedToOutputHandler(t *testing.T) {
	var outputLines []string
	var controlPrefixesGot []string
	controlDone := make(chan struct{}, 5)

	r := NewClaudeCodeRuntime(
		func(line string) { outputLines = append(outputLines, line) },
		func(prefix string, _ json.RawMessage) {
			controlPrefixesGot = append(controlPrefixesGot, prefix)
			controlDone <- struct{}{}
		},
	)

	pr, pw := io.Pipe()
	go func() {
		fmt.Fprintln(pw, "CONDUCTOR_HEARTBEAT {}")
		fmt.Fprintln(pw, "some regular output line")
		fmt.Fprintln(pw, `{"type":"usage","usage":{"input_tokens":5,"output_tokens":3}}`)
		pw.Close()
	}()

	r.readOutput(pr)

	// Wait for the async control goroutine
	select {
	case <-controlDone:
	case <-time.After(200 * time.Millisecond):
		t.Error("control handler goroutine did not fire")
	}

	// Control line should NOT appear in output
	for _, l := range outputLines {
		if l == "CONDUCTOR_HEARTBEAT {}" {
			t.Error("control line should not be forwarded to OutputHandler")
		}
	}

	// Usage line IS forwarded (not a control prefix)
	found := false
	for _, l := range outputLines {
		if l == `{"type":"usage","usage":{"input_tokens":5,"output_tokens":3}}` {
			found = true
			break
		}
	}
	if !found {
		t.Error("usage JSON line should appear in OutputHandler output")
	}

	// Tokens should be accumulated
	if r.TokensUsed() != 8 {
		t.Errorf("expected 8 tokens, got %d", r.TokensUsed())
	}

	// Control prefix should have fired once
	if len(controlPrefixesGot) != 1 || controlPrefixesGot[0] != ControlHeartbeat {
		t.Errorf("expected one heartbeat control, got %v", controlPrefixesGot)
	}
}

func TestReadOutput_NilOutputHandler_NoOutputLines(t *testing.T) {
	r := NewClaudeCodeRuntime(nil, nil)

	pr, pw := io.Pipe()
	go func() {
		fmt.Fprintln(pw, "some output line")
		pw.Close()
	}()

	// Should not panic when OutputHandler is nil
	r.readOutput(pr)
}

func TestReadOutput_MultipleControlMessages(t *testing.T) {
	fired := make(chan string, 10)
	r := NewClaudeCodeRuntime(nil, func(prefix string, _ json.RawMessage) {
		fired <- prefix
	})

	pr, pw := io.Pipe()
	go func() {
		fmt.Fprintln(pw, `CONDUCTOR_HIRE {"role_title":"x"}`)
		fmt.Fprintln(pw, `CONDUCTOR_DONE {"output_path":"/x"}`)
		pw.Close()
	}()

	r.readOutput(pr)

	var got []string
	for i := 0; i < 2; i++ {
		select {
		case p := <-fired:
			got = append(got, p)
		case <-time.After(300 * time.Millisecond):
			t.Fatalf("control handler goroutine %d did not fire (got so far: %v)", i, got)
		}
	}

	if len(got) != 2 {
		t.Errorf("expected 2 control messages, got %d: %v", len(got), got)
	}
}

func TestReadOutput_TokensAccumulateAcrossLines(t *testing.T) {
	r := NewClaudeCodeRuntime(func(_ string) {}, nil)

	pr, pw := io.Pipe()
	go func() {
		fmt.Fprintln(pw, `{"type":"usage","usage":{"input_tokens":10,"output_tokens":5}}`)
		fmt.Fprintln(pw, `{"type":"usage","usage":{"input_tokens":20,"output_tokens":10}}`)
		pw.Close()
	}()

	r.readOutput(pr)

	if r.TokensUsed() != 45 {
		t.Errorf("expected 45 tokens, got %d", r.TokensUsed())
	}
}
