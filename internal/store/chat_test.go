package store_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v2"

	"conductor/internal/store"
)

var chatSessionColumns = []string{"id", "agent_id", "messages", "started_at"}

// --- GetOrCreateChatSession ---

func TestGetOrCreateChatSession_Success(t *testing.T) {
	mock, db := newMock(t)
	agentID := mustUUID(t)
	sessionID := mustUUID(t)
	ts := fixedTime()

	msgs := []store.ChatMessage{
		{Role: "user", Content: "hello", Timestamp: ts},
	}
	msgsBytes, _ := json.Marshal(msgs)

	// Expect the INSERT (ON CONFLICT DO NOTHING).
	mock.ExpectExec(`INSERT INTO chat_sessions`).
		WithArgs(agentID).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	// Expect the SELECT.
	mock.ExpectQuery(`SELECT id, agent_id, messages, started_at FROM chat_sessions`).
		WithArgs(agentID).
		WillReturnRows(mock.NewRows(chatSessionColumns).
			AddRow(sessionID, agentID, msgsBytes, ts))

	session, err := db.GetOrCreateChatSession(context.Background(), agentID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session.ID != sessionID {
		t.Errorf("expected session ID %v, got %v", sessionID, session.ID)
	}
	if len(session.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(session.Messages))
	}
	if session.Messages[0].Role != "user" {
		t.Errorf("expected role 'user', got %q", session.Messages[0].Role)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestGetOrCreateChatSession_EmptyMessages(t *testing.T) {
	mock, db := newMock(t)
	agentID := mustUUID(t)
	sessionID := mustUUID(t)
	ts := fixedTime()
	emptyMsgsBytes, _ := json.Marshal([]store.ChatMessage{})

	mock.ExpectExec(`INSERT INTO chat_sessions`).
		WithArgs(agentID).
		WillReturnResult(pgxmock.NewResult("INSERT", 0)) // ON CONFLICT DO NOTHING

	mock.ExpectQuery(`SELECT id, agent_id, messages, started_at FROM chat_sessions`).
		WithArgs(agentID).
		WillReturnRows(mock.NewRows(chatSessionColumns).
			AddRow(sessionID, agentID, emptyMsgsBytes, ts))

	session, err := db.GetOrCreateChatSession(context.Background(), agentID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(session.Messages) != 0 {
		t.Errorf("expected 0 messages, got %d", len(session.Messages))
	}
}

func TestGetOrCreateChatSession_InsertError(t *testing.T) {
	mock, db := newMock(t)
	agentID := mustUUID(t)

	mock.ExpectExec(`INSERT INTO chat_sessions`).
		WithArgs(agentID).
		WillReturnError(errors.New("insert error"))

	session, err := db.GetOrCreateChatSession(context.Background(), agentID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if session != nil {
		t.Error("expected nil session on error")
	}
}

func TestGetOrCreateChatSession_SelectError(t *testing.T) {
	mock, db := newMock(t)
	agentID := mustUUID(t)

	mock.ExpectExec(`INSERT INTO chat_sessions`).
		WithArgs(agentID).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	mock.ExpectQuery(`SELECT id, agent_id, messages, started_at FROM chat_sessions`).
		WithArgs(agentID).
		WillReturnError(errors.New("select error"))

	session, err := db.GetOrCreateChatSession(context.Background(), agentID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if session != nil {
		t.Error("expected nil session on select error")
	}
}

func TestGetOrCreateChatSession_InvalidJSON(t *testing.T) {
	mock, db := newMock(t)
	agentID := mustUUID(t)
	sessionID := mustUUID(t)
	ts := fixedTime()

	mock.ExpectExec(`INSERT INTO chat_sessions`).
		WithArgs(agentID).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	// Malformed JSON — store should fall back to empty slice.
	mock.ExpectQuery(`SELECT id, agent_id, messages, started_at FROM chat_sessions`).
		WithArgs(agentID).
		WillReturnRows(mock.NewRows(chatSessionColumns).
			AddRow(sessionID, agentID, []byte("not-json"), ts))

	session, err := db.GetOrCreateChatSession(context.Background(), agentID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session.Messages == nil {
		t.Error("expected non-nil Messages slice after bad JSON")
	}
}

// --- AppendChatMessage ---

func TestAppendChatMessage_Success(t *testing.T) {
	mock, db := newMock(t)
	agentID := mustUUID(t)

	// AppendChatMessage passes $1=json-encoded message, $2=agentID.
	// Use AnyArg() for the JSON string since it contains a timestamp.
	mock.ExpectExec(`UPDATE chat_sessions`).
		WithArgs(pgxmock.AnyArg(), agentID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := db.AppendChatMessage(context.Background(), agentID, "user", "hello world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestAppendChatMessage_AssistantRole(t *testing.T) {
	mock, db := newMock(t)
	agentID := mustUUID(t)

	mock.ExpectExec(`UPDATE chat_sessions`).
		WithArgs(pgxmock.AnyArg(), agentID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := db.AppendChatMessage(context.Background(), agentID, "assistant", "here is my response")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAppendChatMessage_Error(t *testing.T) {
	mock, db := newMock(t)
	agentID := mustUUID(t)

	mock.ExpectExec(`UPDATE chat_sessions`).
		WithArgs(pgxmock.AnyArg(), agentID).
		WillReturnError(errors.New("exec error"))

	err := db.AppendChatMessage(context.Background(), agentID, "user", "hello")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- GetChatHistory ---

func TestGetChatHistory_Success(t *testing.T) {
	mock, db := newMock(t)
	agentID := mustUUID(t)
	ts := fixedTime()

	msgs := []store.ChatMessage{
		{Role: "user", Content: "hi", Timestamp: ts},
		{Role: "assistant", Content: "hello", Timestamp: ts},
	}
	msgsBytes, _ := json.Marshal(msgs)

	mock.ExpectQuery(`SELECT messages FROM chat_sessions`).
		WithArgs(agentID).
		WillReturnRows(mock.NewRows([]string{"messages"}).AddRow(msgsBytes))

	history, err := db.GetChatHistory(context.Background(), agentID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(history) != 2 {
		t.Errorf("expected 2 messages, got %d", len(history))
	}
	if history[0].Role != "user" {
		t.Errorf("expected first role 'user', got %q", history[0].Role)
	}
	if history[1].Content != "hello" {
		t.Errorf("expected second content 'hello', got %q", history[1].Content)
	}
}

func TestGetChatHistory_NoSession(t *testing.T) {
	mock, db := newMock(t)
	agentID := mustUUID(t)

	// No session found — should return empty slice, not error.
	mock.ExpectQuery(`SELECT messages FROM chat_sessions`).
		WithArgs(agentID).
		WillReturnError(errors.New("no rows in result set"))

	history, err := db.GetChatHistory(context.Background(), agentID)
	if err != nil {
		t.Fatalf("expected nil error when no session, got %v", err)
	}
	if len(history) != 0 {
		t.Errorf("expected 0 messages, got %d", len(history))
	}
}

func TestGetChatHistory_InvalidJSON(t *testing.T) {
	mock, db := newMock(t)
	agentID := mustUUID(t)

	mock.ExpectQuery(`SELECT messages FROM chat_sessions`).
		WithArgs(agentID).
		WillReturnRows(mock.NewRows([]string{"messages"}).AddRow([]byte("not-json")))

	history, err := db.GetChatHistory(context.Background(), agentID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Falls back to empty slice on bad JSON.
	if len(history) != 0 {
		t.Errorf("expected 0 messages on bad JSON, got %d", len(history))
	}
}

func TestGetChatHistory_Empty(t *testing.T) {
	mock, db := newMock(t)
	agentID := mustUUID(t)
	emptyBytes, _ := json.Marshal([]store.ChatMessage{})

	mock.ExpectQuery(`SELECT messages FROM chat_sessions`).
		WithArgs(agentID).
		WillReturnRows(mock.NewRows([]string{"messages"}).AddRow(emptyBytes))

	history, err := db.GetChatHistory(context.Background(), agentID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(history) != 0 {
		t.Errorf("expected 0 messages, got %d", len(history))
	}
}
