package store_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pashagolub/pgxmock/v2"

	"conductor/internal/store"
)

// newMock returns a pgxmock pool and a *store.DB backed by it.
func newMock(t *testing.T) (pgxmock.PgxPoolIface, *store.DB) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock pool: %v", err)
	}
	db := &store.DB{Pool: mock}
	return mock, db
}

// mustUUID creates a new UUID, failing the test on error.
func mustUUID(t *testing.T) uuid.UUID {
	t.Helper()
	id, err := uuid.NewRandom()
	if err != nil {
		t.Fatalf("uuid.NewRandom: %v", err)
	}
	return id
}

// fixedTime returns a deterministic time value for test assertions.
func fixedTime() time.Time {
	return time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
}
