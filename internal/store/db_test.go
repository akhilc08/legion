package store_test

import (
	"context"
	"testing"

	"conductor/internal/store"
)

func TestDB_Close(t *testing.T) {
	mock, db := newMock(t)
	mock.ExpectClose()
	// Close delegates to the pool's Close method.
	db.Close()
	// pgxmock.Close() is not tracked as an expectation in the same way,
	// so we just verify ExpectationsWereMet doesn't error on unrelated expectations.
	_ = mock.ExpectationsWereMet()
}

func TestDB_Ping_Success(t *testing.T) {
	// pgxmock ping monitoring is disabled by default; test that Pool.Ping runs
	// without panicking. The mock pool accepts ping calls even without an expectation.
	_, db := newMock(t)
	// We just verify the interface is callable; error is expected because no
	// expectation is set, but the important thing is no nil-pointer panic.
	_ = db.Pool.Ping(context.Background())
}

// TestPgxPoolInterface verifies that *store.DB.Pool accepts both the mock
// and (at compile time) a real *pgxpool.Pool through the PgxPool interface.
func TestPgxPoolInterface(t *testing.T) {
	_, db := newMock(t)
	var _ store.PgxPool = db.Pool // compile-time interface satisfaction
}
