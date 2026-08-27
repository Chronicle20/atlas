package purchaserecord

import (
	"strings"
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

func TestRecordUpsertsAndCounts(t *testing.T) {
	db := testDatabase(t)
	a := uuid.New()
	const accountId = uint32(42)
	const serialNumber = uint32(10000)

	t.Run("first purchase creates", func(t *testing.T) {
		if err := Record(db, a, accountId, serialNumber); err != nil {
			t.Fatalf("Record: %v", err)
		}
		count, err := Get(db, a, accountId, serialNumber)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if count != 1 {
			t.Fatalf("count = %d, want 1", count)
		}
	})

	t.Run("second purchase increments", func(t *testing.T) {
		if err := Record(db, a, accountId, serialNumber); err != nil {
			t.Fatalf("Record: %v", err)
		}
		count, err := Get(db, a, accountId, serialNumber)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if count != 2 {
			t.Fatalf("count = %d, want 2", count)
		}
	})

	t.Run("different serial is separate", func(t *testing.T) {
		const otherSerial = uint32(20000)
		if err := Record(db, a, accountId, otherSerial); err != nil {
			t.Fatalf("Record: %v", err)
		}
		count, err := Get(db, a, accountId, otherSerial)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if count != 1 {
			t.Fatalf("count = %d, want 1", count)
		}

		count, err = Get(db, a, accountId, serialNumber)
		if err != nil {
			t.Fatalf("Get original serial: %v", err)
		}
		if count != 2 {
			t.Fatalf("original serial count = %d, want 2", count)
		}
	})

	t.Run("different account is separate", func(t *testing.T) {
		count, err := Get(db, a, 99, serialNumber)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if count != 0 {
			t.Fatalf("count = %d, want 0", count)
		}
	})

	t.Run("different tenant is separate", func(t *testing.T) {
		count, err := Get(db, uuid.New(), accountId, serialNumber)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if count != 0 {
			t.Fatalf("count = %d, want 0", count)
		}
	})

	t.Run("miss is not an error", func(t *testing.T) {
		count, err := Get(db, a, accountId, 30000)
		if err != nil {
			t.Fatalf("Get: %v, want nil error", err)
		}
		if count != 0 {
			t.Fatalf("count = %d, want 0", count)
		}
	})
}

// TestRecordConflictUpdateIsTableQualified is a regression test for the
// Postgres "column reference \"count\" is ambiguous" (SQLSTATE 42702)
// defect: inside ON CONFLICT ... DO UPDATE, an unqualified column on the
// right-hand side of SET resolves against both the target table and the
// EXCLUDED pseudo-relation. sqlite does not enforce this, so it cannot
// catch a regression by executing the statement -- only by inspecting the
// generated SQL directly.
func TestRecordConflictUpdateIsTableQualified(t *testing.T) {
	db := testDatabase(t)
	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return recordTx(tx, uuid.New(), 42, 10000)
	})

	if !strings.Contains(sql, "cash_purchase_records.count + 1") {
		t.Fatalf("expected ON CONFLICT DO UPDATE to table-qualify count, got SQL: %s", sql)
	}
	if strings.Contains(sql, "SET \"count\"=count + 1") {
		t.Fatalf("DO UPDATE SET right-hand side is unqualified, got SQL: %s", sql)
	}
}
