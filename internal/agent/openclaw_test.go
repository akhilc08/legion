package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"testing"
	"time"

	"conductor/internal/store"
	"github.com/google/uuid"
)

func makeTestIssue() store.Issue {
	return store.Issue{
		ID:          uuid.New(),
		CompanyID:   uuid.New(),
		Title:       "test issue",
		Description: "do the thing",
	}
}

// ── NewOpenClawRuntime ────────────────────────────────────────────────────────

func TestNewOpenClawRuntime_NilHandlers(t *testing.T) {
	r := NewOpenClawRuntime(nil, nil)
	if r == nil {
		t.Fatal("expected non-nil runtime")
	}
}

func TestNewOpenClawRuntime_ChannelsInitialized(t *testing.T) {
	r := NewOpenClawRuntime(nil, nil)
	if r.outputCh == nil {
		t.Error("outputCh should be initialized")
	}
	if r.controlCh == nil {
		t.Error("controlCh should be initialized")
	}
	if r.doneCh == nil {
		t.Error("doneCh should be initialized")
	}
	if r.chatReplyCh == nil {
		t.Error("chatReplyCh should be initialized")
	}
}

// ── PID before spawn ──────────────────────────────────────────────────────────

func TestOpenClawPID_BeforeSpawn(t *testing.T) {
	r := NewOpenClawRuntime(nil, nil)
	if r.PID() != 0 {
		t.Errorf("expected PID 0 before spawn, got %d", r.PID())
	}
}

// ── TokensUsed before spawn ───────────────────────────────────────────────────

func TestOpenClawTokensUsed_BeforeSpawn(t *testing.T) {
	r := NewOpenClawRuntime(nil, nil)
	if r.TokensUsed() != 0 {
		t.Errorf("expected 0 tokens, got %d", r.TokensUsed())
	}
}

// ── Kill with no process ──────────────────────────────────────────────────────

func TestOpenClawKill_NoProcess(t *testing.T) {
	r := NewOpenClawRuntime(nil, nil)
	err := r.Kill(context.Background())
	if err != nil {
		t.Errorf("expected nil error killing idle runtime, got %v", err)
	}
}

// ── Heartbeat with no process ─────────────────────────────────────────────────

func TestOpenClawHeartbeat_NoProcess(t *testing.T) {
	r := NewOpenClawRuntime(nil, nil)
	err := r.Heartbeat(context.Background())
	if err == nil {
		t.Error("expected error from Heartbeat when not running (OpenClaw is different from ClaudeCode)")
	}
}

// ── SendTask with no stdin ────────────────────────────────────────────────────

func TestOpenClawSendTask_NotRunning(t *testing.T) {
	r := NewOpenClawRuntime(nil, nil)
	err := r.SendTask(context.Background(), makeTestIssue())
	if err == nil {
		t.Error("expected error from SendTask when not running")
	}
}

// ── SendChat with no stdin ────────────────────────────────────────────────────

func TestOpenClawSendChat_NotRunning(t *testing.T) {
	r := NewOpenClawRuntime(nil, nil)
	_, err := r.SendChat(context.Background(), "hello")
	if err == nil {
		t.Error("expected error from SendChat when not running")
	}
}

// ── SendChat context cancellation ────────────────────────────────────────────

func TestOpenClawSendChat_ContextCancelled(t *testing.T) {
	r := NewOpenClawRuntime(nil, nil)
	// Wire up a fake stdin. We must drain the write so Fprintf doesn't block.
	pr, pw := io.Pipe()
	r.stdin = pw
	// Drain the pipe in a goroutine so the Fprintf call in SendChat succeeds
	go io.Copy(io.Discard, pr) //nolint
	defer pr.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled

	_, err := r.SendChat(ctx, "hello")
	if err == nil {
		t.Error("expected context error, got nil")
	}
}

// ── tryParseTokens ────────────────────────────────────────────────────────────

func TestOpenClawTryParseTokens_ValidMessage(t *testing.T) {
	r := NewOpenClawRuntime(nil, nil)
	r.tryParseTokens(`{"type":"result","tokens_used":75}`)
	if r.TokensUsed() != 75 {
		t.Errorf("expected 75 tokens, got %d", r.TokensUsed())
	}
}

func TestOpenClawTryParseTokens_ZeroTokens(t *testing.T) {
	r := NewOpenClawRuntime(nil, nil)
	// tokens_used == 0 does not match the condition "msg.Tokens > 0"
	r.tryParseTokens(`{"type":"result","tokens_used":0}`)
	if r.TokensUsed() != 0 {
		t.Errorf("expected 0, got %d", r.TokensUsed())
	}
}

func TestOpenClawTryParseTokens_InvalidJSON(t *testing.T) {
	r := NewOpenClawRuntime(nil, nil)
	r.tryParseTokens("not json")
	if r.TokensUsed() != 0 {
		t.Errorf("expected 0 for invalid JSON, got %d", r.TokensUsed())
	}
}

func TestOpenClawTryParseTokens_Accumulates(t *testing.T) {
	r := NewOpenClawRuntime(nil, nil)
	r.tryParseTokens(`{"tokens_used":10}`)
	r.tryParseTokens(`{"tokens_used":20}`)
	r.tryParseTokens(`{"tokens_used":5}`)
	if r.TokensUsed() != 35 {
		t.Errorf("expected 35 accumulated tokens, got %d", r.TokensUsed())
	}
}

func TestOpenClawTryParseTokens_EmptyString(t *testing.T) {
	r := NewOpenClawRuntime(nil, nil)
	r.tryParseTokens("")
	if r.TokensUsed() != 0 {
		t.Errorf("expected 0, got %d", r.TokensUsed())
	}
}

// ── handleControl ─────────────────────────────────────────────────────────────

func TestOpenClawHandleControl_NilHandler(t *testing.T) {
	r := NewOpenClawRuntime(nil, nil)
	r.handleControl("CONDUCTOR_HEARTBEAT {}")
	// Should not panic
}

func TestOpenClawHandleControl_Heartbeat(t *testing.T) {
	done := make(chan string, 1)
	r := NewOpenClawRuntime(nil, func(prefix string, _ json.RawMessage) {
		done <- prefix
	})
	r.handleControl("CONDUCTOR_HEARTBEAT {}")
	select {
	case p := <-done:
		if p != ControlHeartbeat {
			t.Errorf("expected %q, got %q", ControlHeartbeat, p)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("control handler not called")
	}
}

func TestOpenClawHandleControl_AllPrefixes(t *testing.T) {
	prefixes := []string{
		ControlHire,
		ControlEscalate,
		ControlDone,
		ControlBlocked,
		ControlHeartbeat,
	}
	for _, prefix := range prefixes {
		t.Run(prefix, func(t *testing.T) {
			done := make(chan string, 1)
			r := NewOpenClawRuntime(nil, func(p string, _ json.RawMessage) {
				done <- p
			})
			r.handleControl(prefix + ` {}`)
			select {
			case p := <-done:
				if p != prefix {
					t.Errorf("expected %q, got %q", prefix, p)
				}
			case <-time.After(100 * time.Millisecond):
				t.Fatalf("control handler not called for prefix %q", prefix)
			}
		})
	}
}

func TestOpenClawHandleControl_EmptyPayload_UsesEmptyJSON(t *testing.T) {
	var gotPayload json.RawMessage
	done := make(chan struct{}, 1)
	r := NewOpenClawRuntime(nil, func(_ string, payload json.RawMessage) {
		gotPayload = payload
		done <- struct{}{}
	})
	r.handleControl("CONDUCTOR_DONE")
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("control handler not called")
	}
	if string(gotPayload) != "{}" {
		t.Errorf("expected '{}' for empty payload, got %q", gotPayload)
	}
}

// ── readStdout ────────────────────────────────────────────────────────────────

func TestOpenClawReadStdout_ControlGoesToControlCh(t *testing.T) {
	r := NewOpenClawRuntime(nil, nil)

	pr, pw := io.Pipe()
	go func() {
		fmt.Fprintln(pw, "CONDUCTOR_HEARTBEAT {}")
		pw.Close()
	}()

	r.readStdout(pr)

	select {
	case line := <-r.controlCh:
		if line != "CONDUCTOR_HEARTBEAT {}" {
			t.Errorf("expected control line, got %q", line)
		}
	default:
		t.Error("expected control line in controlCh")
	}
}

func TestOpenClawReadStdout_OutputGoesToOutputCh(t *testing.T) {
	r := NewOpenClawRuntime(nil, nil)

	pr, pw := io.Pipe()
	go func() {
		fmt.Fprintln(pw, "regular output line")
		pw.Close()
	}()

	r.readStdout(pr)

	select {
	case line := <-r.outputCh:
		if line != "regular output line" {
			t.Errorf("expected output line, got %q", line)
		}
	default:
		t.Error("expected output line in outputCh")
	}
}

func TestOpenClawReadStdout_ControlNotInOutputCh(t *testing.T) {
	r := NewOpenClawRuntime(nil, nil)

	pr, pw := io.Pipe()
	go func() {
		fmt.Fprintln(pw, "CONDUCTOR_DONE {}")
		pw.Close()
	}()

	r.readStdout(pr)

	select {
	case line := <-r.outputCh:
		t.Errorf("control line should not appear in outputCh, got %q", line)
	default:
		// Good — nothing in outputCh
	}
}

// ── dispatchOutput ────────────────────────────────────────────────────────────

func TestOpenClawDispatchOutput_RoutesToOutputHandler(t *testing.T) {
	var gotLines []string
	done := make(chan struct{})
	r := NewOpenClawRuntime(
		func(line string) {
			gotLines = append(gotLines, line)
			close(done)
		},
		nil,
	)

	// Send one line and close the channel
	go func() {
		r.outputCh <- "hello from agent"
		// Close both channels to let dispatchOutput return
		close(r.outputCh)
	}()

	r.dispatchOutput()

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("output handler never called")
	}
	if len(gotLines) == 0 || gotLines[0] != "hello from agent" {
		t.Errorf("expected 'hello from agent', got %v", gotLines)
	}
}

func TestOpenClawDispatchOutput_ParsesTokensFromOutputLines(t *testing.T) {
	done := make(chan struct{})
	r := NewOpenClawRuntime(
		func(_ string) { close(done) },
		nil,
	)

	go func() {
		r.outputCh <- `{"tokens_used":42}`
		close(r.outputCh)
	}()

	r.dispatchOutput()

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("output handler never called")
	}

	if r.TokensUsed() != 42 {
		t.Errorf("expected 42 tokens, got %d", r.TokensUsed())
	}
}

func TestOpenClawDispatchOutput_WhenPaused_RoutesToChatReply(t *testing.T) {
	r := NewOpenClawRuntime(nil, nil)
	r.mu.Lock()
	r.paused = true
	r.mu.Unlock()

	done := make(chan struct{})
	go func() {
		r.outputCh <- "agent reply"
		close(r.outputCh)
		close(done)
	}()

	r.dispatchOutput()
	<-done

	select {
	case reply := <-r.chatReplyCh:
		if reply != "agent reply" {
			t.Errorf("expected 'agent reply', got %q", reply)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("expected reply in chatReplyCh when paused")
	}
}

func TestOpenClawDispatchOutput_ControlLineInvokesHandler(t *testing.T) {
	handlerDone := make(chan string, 1)
	r := NewOpenClawRuntime(nil, func(prefix string, _ json.RawMessage) {
		handlerDone <- prefix
	})

	go func() {
		r.controlCh <- "CONDUCTOR_HEARTBEAT {}"
		close(r.controlCh)
	}()

	r.dispatchOutput()

	select {
	case p := <-handlerDone:
		if p != ControlHeartbeat {
			t.Errorf("expected heartbeat prefix, got %q", p)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("control handler not called")
	}
}
