package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// Log appends an immutable audit event.
func (db *DB) Log(ctx context.Context, companyID uuid.UUID, actorID *uuid.UUID, eventType string, payload map[string]interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal audit payload: %w", err)
	}
	_, err = db.Pool.Exec(ctx,
		`INSERT INTO audit_log (company_id, actor_id, event_type, payload) VALUES ($1,$2,$3,$4)`,
		companyID, actorID, eventType, data,
	)
	return err
}

func (db *DB) ListAuditLog(ctx context.Context, companyID uuid.UUID, limit int) ([]AuditLog, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := db.Pool.Query(ctx,
		`SELECT id, company_id, actor_id, event_type, payload, created_at
		 FROM audit_log WHERE company_id = $1 ORDER BY created_at DESC LIMIT $2`,
		companyID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list audit log: %w", err)
	}
	defer rows.Close()

	var logs []AuditLog
	for rows.Next() {
		var l AuditLog
		var payloadBytes []byte
		if err := rows.Scan(&l.ID, &l.CompanyID, &l.ActorID, &l.EventType, &payloadBytes, &l.CreatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(payloadBytes, &l.Payload); err != nil {
			l.Payload = map[string]interface{}{}
		}
		logs = append(logs, l)
	}
	return logs, nil
}

// Notification store methods

func (db *DB) CreateNotification(ctx context.Context, n *Notification) (*Notification, error) {
	data, _ := json.Marshal(n.Payload)
	out := &Notification{}
	var payloadBytes []byte
	err := db.Pool.QueryRow(ctx,
		`INSERT INTO notifications (company_id, type, escalation_id, payload)
		 VALUES ($1,$2,$3,$4)
		 RETURNING id, company_id, type, escalation_id, payload, dismissed_at, created_at`,
		n.CompanyID, n.Type, n.EscalationID, data,
	).Scan(&out.ID, &out.CompanyID, &out.Type, &out.EscalationID, &payloadBytes, &out.DismissedAt, &out.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create notification: %w", err)
	}
	json.Unmarshal(payloadBytes, &out.Payload) //nolint
	return out, nil
}

func (db *DB) ListActiveNotifications(ctx context.Context, companyID uuid.UUID) ([]Notification, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT id, company_id, type, escalation_id, payload, dismissed_at, created_at
		 FROM notifications WHERE company_id = $1 AND dismissed_at IS NULL ORDER BY created_at DESC`,
		companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()

	var ns []Notification
	for rows.Next() {
		var n Notification
		var payloadBytes []byte
		if err := rows.Scan(&n.ID, &n.CompanyID, &n.Type, &n.EscalationID, &payloadBytes, &n.DismissedAt, &n.CreatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal(payloadBytes, &n.Payload) //nolint
		ns = append(ns, n)
	}
	return ns, nil
}

func (db *DB) DismissNotification(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx,
		`UPDATE notifications SET dismissed_at = NOW() WHERE id = $1`, id,
	)
	return err
}
