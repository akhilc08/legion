package store_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/pashagolub/pgxmock/v2"

	"conductor/internal/store"
)

// --- Log (audit) ---

func TestLog_Success(t *testing.T) {
	mock, db := newMock(t)
	companyID := mustUUID(t)
	actorID := mustUUID(t)
	payload := map[string]interface{}{"action": "login"}
	payloadBytes, _ := json.Marshal(payload)

	mock.ExpectExec(`INSERT INTO audit_log`).
		WithArgs(companyID, &actorID, "user.login", payloadBytes).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err := db.Log(context.Background(), companyID, &actorID, "user.login", payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestLog_NilActor(t *testing.T) {
	mock, db := newMock(t)
	companyID := mustUUID(t)
	payload := map[string]interface{}{"system": "startup"}
	payloadBytes, _ := json.Marshal(payload)

	mock.ExpectExec(`INSERT INTO audit_log`).
		WithArgs(companyID, (*uuid.UUID)(nil), "system.startup", payloadBytes).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err := db.Log(context.Background(), companyID, nil, "system.startup", payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLog_DBError(t *testing.T) {
	mock, db := newMock(t)
	companyID := mustUUID(t)

	mock.ExpectExec(`INSERT INTO audit_log`).
		WillReturnError(errors.New("db error"))

	err := db.Log(context.Background(), companyID, nil, "event", map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLog_EmptyPayload(t *testing.T) {
	mock, db := newMock(t)
	companyID := mustUUID(t)
	payload := map[string]interface{}{}
	payloadBytes, _ := json.Marshal(payload)

	mock.ExpectExec(`INSERT INTO audit_log`).
		WithArgs(companyID, (*uuid.UUID)(nil), "ping", payloadBytes).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err := db.Log(context.Background(), companyID, nil, "ping", payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- ListAuditLog ---

var auditColumns = []string{"id", "company_id", "actor_id", "event_type", "payload", "created_at"}

func TestListAuditLog_Success(t *testing.T) {
	mock, db := newMock(t)
	companyID := mustUUID(t)
	logID := mustUUID(t)
	actorID := mustUUID(t)
	ts := fixedTime()
	payloadBytes, _ := json.Marshal(map[string]interface{}{"key": "val"})

	mock.ExpectQuery(`SELECT .* FROM audit_log WHERE company_id`).
		WithArgs(companyID, 50).
		WillReturnRows(mock.NewRows(auditColumns).
			AddRow(logID, companyID, &actorID, "user.created", payloadBytes, ts))

	logs, err := db.ListAuditLog(context.Background(), companyID, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(logs) != 1 {
		t.Errorf("expected 1 log, got %d", len(logs))
	}
	if logs[0].EventType != "user.created" {
		t.Errorf("expected 'user.created', got %q", logs[0].EventType)
	}
	if logs[0].Payload["key"] != "val" {
		t.Errorf("expected payload key=val, got %v", logs[0].Payload)
	}
}

func TestListAuditLog_DefaultLimit(t *testing.T) {
	mock, db := newMock(t)
	companyID := mustUUID(t)

	// Limit <= 0 should default to 100.
	mock.ExpectQuery(`SELECT .* FROM audit_log WHERE company_id`).
		WithArgs(companyID, 100).
		WillReturnRows(mock.NewRows(auditColumns))

	_, err := db.ListAuditLog(context.Background(), companyID, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestListAuditLog_NegativeLimit(t *testing.T) {
	mock, db := newMock(t)
	companyID := mustUUID(t)

	mock.ExpectQuery(`SELECT .* FROM audit_log WHERE company_id`).
		WithArgs(companyID, 100).
		WillReturnRows(mock.NewRows(auditColumns))

	_, err := db.ListAuditLog(context.Background(), companyID, -5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListAuditLog_QueryError(t *testing.T) {
	mock, db := newMock(t)
	companyID := mustUUID(t)

	mock.ExpectQuery(`SELECT .* FROM audit_log WHERE company_id`).
		WillReturnError(errors.New("query error"))

	_, err := db.ListAuditLog(context.Background(), companyID, 10)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestListAuditLog_InvalidPayloadJSON(t *testing.T) {
	mock, db := newMock(t)
	companyID := mustUUID(t)
	logID := mustUUID(t)
	ts := fixedTime()

	// Malformed JSON payload — store should handle gracefully by returning empty map.
	mock.ExpectQuery(`SELECT .* FROM audit_log WHERE company_id`).
		WithArgs(companyID, 10).
		WillReturnRows(mock.NewRows(auditColumns).
			AddRow(logID, companyID, nil, "event", []byte("not-json"), ts))

	logs, err := db.ListAuditLog(context.Background(), companyID, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}
	// Should be empty map, not nil.
	if logs[0].Payload == nil {
		t.Error("expected non-nil payload map after JSON parse failure")
	}
}

// --- CreateNotification ---

var notificationColumns = []string{
	"id", "company_id", "type", "escalation_id", "payload", "dismissed_at", "created_at",
}

func sampleNotification(t *testing.T, companyID uuid.UUID) *store.Notification {
	t.Helper()
	return &store.Notification{
		CompanyID:    companyID,
		Type:         "escalation",
		EscalationID: nil,
		Payload:      map[string]interface{}{"msg": "alert"},
	}
}

func TestCreateNotification_Success(t *testing.T) {
	mock, db := newMock(t)
	companyID := mustUUID(t)
	notifID := mustUUID(t)
	ts := fixedTime()
	n := sampleNotification(t, companyID)
	payloadBytes, _ := json.Marshal(n.Payload)

	mock.ExpectQuery(`INSERT INTO notifications`).
		WithArgs(companyID, "escalation", (*uuid.UUID)(nil), payloadBytes).
		WillReturnRows(mock.NewRows(notificationColumns).
			AddRow(notifID, companyID, "escalation", nil, payloadBytes, nil, ts))

	notif, err := db.CreateNotification(context.Background(), n)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notif.ID != notifID {
		t.Errorf("expected ID %v, got %v", notifID, notif.ID)
	}
}

func TestCreateNotification_Error(t *testing.T) {
	mock, db := newMock(t)
	companyID := mustUUID(t)
	n := sampleNotification(t, companyID)

	mock.ExpectQuery(`INSERT INTO notifications`).
		WillReturnError(errors.New("db error"))

	notif, err := db.CreateNotification(context.Background(), n)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if notif != nil {
		t.Error("expected nil notification on error")
	}
}

// --- ListActiveNotifications ---

func TestListActiveNotifications_Success(t *testing.T) {
	mock, db := newMock(t)
	companyID := mustUUID(t)
	notifID := mustUUID(t)
	ts := fixedTime()
	payloadBytes, _ := json.Marshal(map[string]interface{}{"x": 1})

	mock.ExpectQuery(`SELECT .* FROM notifications WHERE company_id .* AND dismissed_at IS NULL`).
		WithArgs(companyID).
		WillReturnRows(mock.NewRows(notificationColumns).
			AddRow(notifID, companyID, "info", nil, payloadBytes, nil, ts))

	notifs, err := db.ListActiveNotifications(context.Background(), companyID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notifs) != 1 {
		t.Errorf("expected 1 notification, got %d", len(notifs))
	}
}

func TestListActiveNotifications_Empty(t *testing.T) {
	mock, db := newMock(t)
	companyID := mustUUID(t)

	mock.ExpectQuery(`SELECT .* FROM notifications WHERE company_id .* AND dismissed_at IS NULL`).
		WithArgs(companyID).
		WillReturnRows(mock.NewRows(notificationColumns))

	notifs, err := db.ListActiveNotifications(context.Background(), companyID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notifs) != 0 {
		t.Errorf("expected 0 notifications, got %d", len(notifs))
	}
}

func TestListActiveNotifications_Error(t *testing.T) {
	mock, db := newMock(t)
	companyID := mustUUID(t)

	mock.ExpectQuery(`SELECT .* FROM notifications WHERE company_id .* AND dismissed_at IS NULL`).
		WillReturnError(errors.New("query error"))

	_, err := db.ListActiveNotifications(context.Background(), companyID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- DismissNotification ---

func TestDismissNotification_Success(t *testing.T) {
	mock, db := newMock(t)
	id := mustUUID(t)

	mock.ExpectExec(`UPDATE notifications SET dismissed_at`).
		WithArgs(id).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := db.DismissNotification(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDismissNotification_Error(t *testing.T) {
	mock, db := newMock(t)
	id := mustUUID(t)

	mock.ExpectExec(`UPDATE notifications SET dismissed_at`).
		WillReturnError(errors.New("update error"))

	err := db.DismissNotification(context.Background(), id)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
