package opening

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
)

// testDatabase returns a fresh sqlite-in-memory *gorm.DB migrated for this
// package, via the module's established in-memory test-database helper.
func testDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	return databasetest.NewInMemoryTenantDB(t, Migration)
}

func TestInsertIsIdempotentOnTransactionId(t *testing.T) {
	db := testDatabase(t)
	tenantId, txId := uuid.New(), uuid.New()

	if err := Insert(db, tenantId, txId, 10, 100); err != nil {
		t.Fatalf("first insert: %v", err)
	}
	err := Insert(db, tenantId, txId, 10, 100)
	if !errors.Is(err, ErrAlreadyOpened) {
		t.Fatalf("second insert err = %v, want ErrAlreadyOpened", err)
	}
}

// The primary key is (tenant_id, transaction_id): the SAME transaction id
// belonging to a different tenant is a different opening.
func TestInsertIsScopedByTenant(t *testing.T) {
	db := testDatabase(t)
	txId := uuid.New()
	if err := Insert(db, uuid.New(), txId, 10, 100); err != nil {
		t.Fatalf("tenant A: %v", err)
	}
	if err := Insert(db, uuid.New(), txId, 10, 100); err != nil {
		t.Fatalf("tenant B must not collide with tenant A: %v", err)
	}
}

func TestInsertAllowsDistinctTransactions(t *testing.T) {
	db := testDatabase(t)
	tenantId := uuid.New()
	if err := Insert(db, tenantId, uuid.New(), 10, 100); err != nil {
		t.Fatalf("first click: %v", err)
	}
	// A genuine second click mints a new transaction id and must succeed.
	if err := Insert(db, tenantId, uuid.New(), 10, 100); err != nil {
		t.Fatalf("second click: %v", err)
	}
}
