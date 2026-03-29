package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

func (db *DB) GetOrCreateChatSession(ctx context.Context, agentID uuid.UUID) (*ChatSession, error) {
	_, err := db.Pool.Exec(ctx,
		`INSERT INTO chat_sessions (agent_id, messages)
		 VALUES ($1, '[]'::jsonb)
		 ON CONFLICT DO NOTHING`,
		agentID,
	)
	if err != nil {
		return nil, fmt.Errorf("insert chat session: %w", err)
	}

	var s ChatSession
	var messagesRaw []byte
	err = db.Pool.QueryRow(ctx,
		`SELECT id, agent_id, messages, started_at
		 FROM chat_sessions
		 WHERE agent_id = $1
		 ORDER BY started_at DESC
		 LIMIT 1`,
		agentID,
	).Scan(&s.ID, &s.AgentID, &messagesRaw, &s.StartedAt)
	if err != nil {
		return nil, fmt.Errorf("get chat session: %w", err)
	}

	if err := json.Unmarshal(messagesRaw, &s.Messages); err != nil {
		s.Messages = []ChatMessage{}
	}
	return &s, nil
}

func (db *DB) AppendChatMessage(ctx context.Context, agentID uuid.UUID, role, content string) error {
	msg := ChatMessage{
		Role:      role,
		Content:   content,
		Timestamp: time.Now().UTC(),
	}
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal chat message: %w", err)
	}

	_, err = db.Pool.Exec(ctx,
		`UPDATE chat_sessions
		 SET messages = messages || $1::jsonb
		 WHERE id = (
		   SELECT id FROM chat_sessions
		   WHERE agent_id = $2
		   ORDER BY started_at DESC
		   LIMIT 1
		 )`,
		string(msgBytes), agentID,
	)
	if err != nil {
		return fmt.Errorf("append chat message: %w", err)
	}
	return nil
}

func (db *DB) GetChatHistory(ctx context.Context, agentID uuid.UUID) ([]ChatMessage, error) {
	var messagesRaw []byte
	err := db.Pool.QueryRow(ctx,
		`SELECT messages
		 FROM chat_sessions
		 WHERE agent_id = $1
		 ORDER BY started_at DESC
		 LIMIT 1`,
		agentID,
	).Scan(&messagesRaw)
	if err != nil {
		// No session yet — return empty slice
		return []ChatMessage{}, nil
	}

	var msgs []ChatMessage
	if err := json.Unmarshal(messagesRaw, &msgs); err != nil {
		return []ChatMessage{}, nil
	}
	return msgs, nil
}
