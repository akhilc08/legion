// Package stdout provides utilities for parsing conductor control messages
// from agent stdout streams. Control messages are prefixed lines containing
// JSON payloads that trigger orchestrator actions.
package stdout

import (
	"encoding/json"
	"fmt"
	"strings"

	"conductor/internal/agent"
)

// ParsedControl represents a fully decoded control message.
type ParsedControl struct {
	Prefix  string
	Payload json.RawMessage
}

// IsControlLine returns true if the line starts with a known conductor prefix.
func IsControlLine(line string) bool {
	for _, p := range []string{
		agent.ControlHire,
		agent.ControlEscalate,
		agent.ControlDone,
		agent.ControlBlocked,
		agent.ControlHeartbeat,
	} {
		if strings.HasPrefix(line, p) {
			return true
		}
	}
	return false
}

// ParseLine extracts prefix and JSON payload from a control line.
func ParseLine(line string) (*ParsedControl, error) {
	for _, p := range []string{
		agent.ControlHire,
		agent.ControlEscalate,
		agent.ControlDone,
		agent.ControlBlocked,
		agent.ControlHeartbeat,
	} {
		if strings.HasPrefix(line, p) {
			rest := strings.TrimSpace(strings.TrimPrefix(line, p))
			raw := json.RawMessage("{}")
			if len(rest) > 0 {
				if !json.Valid([]byte(rest)) {
					return nil, fmt.Errorf("invalid JSON in control message %s: %s", p, rest)
				}
				raw = json.RawMessage(rest)
			}
			return &ParsedControl{Prefix: p, Payload: raw}, nil
		}
	}
	return nil, fmt.Errorf("not a control line")
}

// DecodeHire parses a CONDUCTOR_HIRE payload.
func DecodeHire(raw json.RawMessage) (*agent.HirePayload, error) {
	var h agent.HirePayload
	if err := json.Unmarshal(raw, &h); err != nil {
		return nil, fmt.Errorf("decode hire: %w", err)
	}
	if h.RoleTitle == "" {
		return nil, fmt.Errorf("hire: role_title required")
	}
	if h.SystemPrompt == "" {
		return nil, fmt.Errorf("hire: system_prompt required")
	}
	if h.BudgetAllocation <= 0 {
		h.BudgetAllocation = 50000
	}
	return &h, nil
}

// DecodeEscalate parses a CONDUCTOR_ESCALATE payload.
func DecodeEscalate(raw json.RawMessage) (*agent.EscalatePayload, error) {
	var e agent.EscalatePayload
	if err := json.Unmarshal(raw, &e); err != nil {
		return nil, fmt.Errorf("decode escalate: %w", err)
	}
	return &e, nil
}

// DecodeDone parses a CONDUCTOR_DONE payload.
func DecodeDone(raw json.RawMessage) (*agent.DonePayload, error) {
	var d agent.DonePayload
	if err := json.Unmarshal(raw, &d); err != nil {
		return nil, fmt.Errorf("decode done: %w", err)
	}
	return &d, nil
}

// DecodeBlocked parses a CONDUCTOR_BLOCKED payload.
func DecodeBlocked(raw json.RawMessage) (*agent.BlockedPayload, error) {
	var b agent.BlockedPayload
	if err := json.Unmarshal(raw, &b); err != nil {
		return nil, fmt.Errorf("decode blocked: %w", err)
	}
	return &b, nil
}
