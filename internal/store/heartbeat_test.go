package store_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v2"

	"conductor/internal/store"
)

// --- UpsertHeartbeat ---

func TestUpsertHeartbeat_Success(t *testing.T) {
	mock, db := newMock(t)
	agentID := mustUUID(t)

	mock.ExpectExec(`INSERT INTO heartbeats`).
		WithArgs(agentID).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err := db.UpsertHeartbeat(context.Background(), agentID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestUpsertHeartbeat_Upsert(t *testing.T) {
	mock, db := newMock(t)
	agentID := mustUUID(t)

	// ON CONFLICT DO UPDATE: simulate 0 rows inserted (update path).
	mock.ExpectExec(`INSERT INTO heartbeats`).
		WithArgs(agentID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err := db.UpsertHeartbeat(context.Background(), agentID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpsertHeartbeat_Error(t *testing.T) {
	mock, db := newMock(t)
	agentID := mustUUID(t)

	mock.ExpectExec(`INSERT INTO heartbeats`).
		WillReturnError(errors.New("exec error"))

	err := db.UpsertHeartbeat(context.Background(), agentID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- GetHeartbeat ---

var heartbeatColumns = []string{"agent_id", "last_seen_at", "consecutive_misses"}

func TestGetHeartbeat_Success(t *testing.T) {
	mock, db := newMock(t)
	agentID := mustUUID(t)
	ts := fixedTime()

	mock.ExpectQuery(`SELECT agent_id, last_seen_at, consecutive_misses FROM heartbeats`).
		WithArgs(agentID).
		WillReturnRows(mock.NewRows(heartbeatColumns).AddRow(agentID, ts, 0))

	hb, err := db.GetHeartbeat(context.Background(), agentID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hb.AgentID != agentID {
		t.Errorf("expected AgentID %v, got %v", agentID, hb.AgentID)
	}
	if hb.ConsecutiveMisses != 0 {
		t.Errorf("expected 0 misses, got %d", hb.ConsecutiveMisses)
	}
}

func TestGetHeartbeat_NotFound(t *testing.T) {
	mock, db := newMock(t)
	agentID := mustUUID(t)

	mock.ExpectQuery(`SELECT agent_id, last_seen_at, consecutive_misses FROM heartbeats`).
		WithArgs(agentID).
		WillReturnError(errors.New("no rows"))

	hb, err := db.GetHeartbeat(context.Background(), agentID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if hb != nil {
		t.Error("expected nil heartbeat")
	}
}

// --- IncrementMisses ---

func TestIncrementMisses_Success(t *testing.T) {
	mock, db := newMock(t)
	agentID := mustUUID(t)

	mock.ExpectQuery(`INSERT INTO heartbeats .* ON CONFLICT .* DO UPDATE .* RETURNING consecutive_misses`).
		WithArgs(agentID).
		WillReturnRows(mock.NewRows([]string{"consecutive_misses"}).AddRow(3))

	misses, err := db.IncrementMisses(context.Background(), agentID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if misses != 3 {
		t.Errorf("expected 3 misses, got %d", misses)
	}
}

func TestIncrementMisses_FirstMiss(t *testing.T) {
	mock, db := newMock(t)
	agentID := mustUUID(t)

	mock.ExpectQuery(`INSERT INTO heartbeats .* ON CONFLICT .* DO UPDATE .* RETURNING consecutive_misses`).
		WithArgs(agentID).
		WillReturnRows(mock.NewRows([]string{"consecutive_misses"}).AddRow(1))

	misses, err := db.IncrementMisses(context.Background(), agentID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if misses != 1 {
		t.Errorf("expected 1 miss, got %d", misses)
	}
}

func TestIncrementMisses_Error(t *testing.T) {
	mock, db := newMock(t)
	agentID := mustUUID(t)

	mock.ExpectQuery(`INSERT INTO heartbeats .* ON CONFLICT .* DO UPDATE .* RETURNING consecutive_misses`).
		WillReturnError(errors.New("query error"))

	_, err := db.IncrementMisses(context.Background(), agentID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- StaleAgents ---

func TestStaleAgents_Success(t *testing.T) {
	mock, db := newMock(t)
	a := sampleAgent(t)
	threshold := 5 * time.Minute

	// The cutoff timestamp is computed inside StaleAgents, so use AnyArg().
	mock.ExpectQuery(`SELECT a\.id, a\.company_id`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(agentRow(mock, a))

	agents, err := db.StaleAgents(context.Background(), threshold)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(agents) != 1 {
		t.Errorf("expected 1 stale agent, got %d", len(agents))
	}
	if agents[0].ID != a.ID {
		t.Errorf("expected ID %v, got %v", a.ID, agents[0].ID)
	}
}

func TestStaleAgents_Empty(t *testing.T) {
	mock, db := newMock(t)
	threshold := time.Minute

	mock.ExpectQuery(`SELECT a\.id, a\.company_id`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(mock.NewRows(agentColumns))

	agents, err := db.StaleAgents(context.Background(), threshold)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(agents) != 0 {
		t.Errorf("expected 0 stale agents, got %d", len(agents))
	}
}

func TestStaleAgents_QueryError(t *testing.T) {
	mock, db := newMock(t)

	mock.ExpectQuery(`SELECT a\.id, a\.company_id`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnError(errors.New("query error"))

	_, err := db.StaleAgents(context.Background(), time.Minute)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// Verify Heartbeat struct fields.
var _ = store.Heartbeat{
	ConsecutiveMisses: 0,
}
