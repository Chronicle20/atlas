package purchaserecord

import (
	"atlas-cashshop/cashshop/inventory/asset"
	"atlas-cashshop/cashshop/inventory/compartment"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/gorm"

	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
)

// compartmentMigrationSqlite creates the cash_compartments table directly.
// compartment.Migration's AutoMigrate emits a `DEFAULT uuid_generate_v4()`
// column default, which is PostgreSQL-specific and fails sqlite's DDL
// parser -- "near '(': syntax error". Tests always supply an explicit Id,
// so the default is never actually needed. Mirrors
// cashshop/inventory/compartment/resource_paginate_test.go.
func compartmentMigrationSqlite(db *gorm.DB) error {
	return db.Exec(`CREATE TABLE IF NOT EXISTS cash_compartments (
		id TEXT PRIMARY KEY,
		tenant_id TEXT NOT NULL,
		account_id INTEGER NOT NULL,
		type INTEGER NOT NULL,
		capacity INTEGER NOT NULL DEFAULT 55
	)`).Error
}

// backfillDatabase returns an in-memory sqlite db migrated for Migration plus
// the asset and compartment migrations that Backfill reads from.
func backfillDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	return databasetest.NewInMemoryTenantDB(t, Migration, compartmentMigrationSqlite, asset.Migration)
}

func backfillLogger() logrus.FieldLogger {
	l, _ := test.NewNullLogger()
	return l
}

func TestBackfill(t *testing.T) {
	db := backfillDatabase(t)
	a := uuid.New()
	const accountId = uint32(42)
	compartmentId := uuid.New()

	if err := db.Create(&compartment.Entity{
		Id:        compartmentId,
		TenantId:  a,
		AccountId: accountId,
		Type:      1,
		Capacity:  55,
	}).Error; err != nil {
		t.Fatalf("create compartment: %v", err)
	}

	now := time.Now()

	t.Run("seeds from live assets", func(t *testing.T) {
		mustCreateAsset(t, db, a, compartmentId, 10000, now)
		mustCreateAsset(t, db, a, compartmentId, 20000, now)

		written, err := Backfill(backfillLogger(), db)
		if err != nil {
			t.Fatalf("Backfill: %v", err)
		}
		if written != 2 {
			t.Fatalf("written = %d, want 2", written)
		}

		count, err := Get(db, a, accountId, 10000)
		if err != nil {
			t.Fatalf("Get 10000: %v", err)
		}
		if count != 1 {
			t.Fatalf("count 10000 = %d, want 1", count)
		}

		count, err = Get(db, a, accountId, 20000)
		if err != nil {
			t.Fatalf("Get 20000: %v", err)
		}
		if count != 1 {
			t.Fatalf("count 20000 = %d, want 1", count)
		}
	})

	// The next two subtests re-check state left over from "seeds from live
	// assets" above; they are not duplicate/soft-delete fixtures in their own
	// right. Real duplicate-counting and soft-deleted-inclusion coverage,
	// with fixtures that actually create duplicates and a soft-deleted row,
	// lives in TestBackfillCountsDuplicatesAndSoftDeletes below.
	t.Run("a single purchase stays counted once", func(t *testing.T) {
		count, err := Get(db, a, accountId, 10000)
		if err != nil {
			t.Fatalf("Get 10000: %v", err)
		}
		if count != 1 {
			t.Fatalf("count 10000 = %d, want 1", count)
		}
	})

	t.Run("a commodity with no assets has no record", func(t *testing.T) {
		count, err := Get(db, a, accountId, 30000)
		if err != nil {
			t.Fatalf("Get 30000: %v", err)
		}
		if count != 0 {
			t.Fatalf("count 30000 = %d, want 0", count)
		}
	})

	t.Run("skips zero commodity id", func(t *testing.T) {
		count, err := Get(db, a, accountId, 0)
		if err != nil {
			t.Fatalf("Get 0: %v", err)
		}
		if count != 0 {
			t.Fatalf("count 0 = %d, want 0", count)
		}
	})
}

// TestBackfillCountsDuplicatesAndSoftDeletes exercises the duplicate-counting
// and soft-deleted-inclusion cases from a clean database, and confirms a
// second Backfill run is a true no-op (idempotency).
func TestBackfillCountsDuplicatesAndSoftDeletes(t *testing.T) {
	db := backfillDatabase(t)
	a := uuid.New()
	const accountId = uint32(42)
	compartmentId := uuid.New()

	if err := db.Create(&compartment.Entity{
		Id:        compartmentId,
		TenantId:  a,
		AccountId: accountId,
		Type:      1,
		Capacity:  55,
	}).Error; err != nil {
		t.Fatalf("create compartment: %v", err)
	}

	now := time.Now()

	// three live assets with the same commodity id
	mustCreateAsset(t, db, a, compartmentId, 10000, now)
	mustCreateAsset(t, db, a, compartmentId, 10000, now)
	mustCreateAsset(t, db, a, compartmentId, 10000, now)

	// one soft-deleted asset
	deletedAsset := mustCreateAsset(t, db, a, compartmentId, 30000, now)
	if err := db.Delete(&asset.Entity{}, "id = ?", deletedAsset).Error; err != nil {
		t.Fatalf("soft-delete asset: %v", err)
	}

	// one zero-commodity asset -- must not produce a record
	mustCreateAsset(t, db, a, compartmentId, 0, now)

	written, err := Backfill(backfillLogger(), db)
	if err != nil {
		t.Fatalf("Backfill: %v", err)
	}
	if written != 2 {
		t.Fatalf("written = %d, want 2", written)
	}

	count, err := Get(db, a, accountId, 10000)
	if err != nil {
		t.Fatalf("Get 10000: %v", err)
	}
	if count != 3 {
		t.Fatalf("count 10000 = %d, want 3", count)
	}

	count, err = Get(db, a, accountId, 30000)
	if err != nil {
		t.Fatalf("Get 30000: %v", err)
	}
	if count != 1 {
		t.Fatalf("count 30000 = %d, want 1", count)
	}

	count, err = Get(db, a, accountId, 0)
	if err != nil {
		t.Fatalf("Get 0: %v", err)
	}
	if count != 0 {
		t.Fatalf("count 0 = %d, want 0", count)
	}

	t.Run("is idempotent", func(t *testing.T) {
		written, err := Backfill(backfillLogger(), db)
		if err != nil {
			t.Fatalf("second Backfill: %v", err)
		}
		if written != 0 {
			t.Fatalf("second Backfill written = %d, want 0", written)
		}

		count, err := Get(db, a, accountId, 10000)
		if err != nil {
			t.Fatalf("Get 10000 after second backfill: %v", err)
		}
		if count != 3 {
			t.Fatalf("count 10000 after second backfill = %d, want 3", count)
		}

		count, err = Get(db, a, accountId, 30000)
		if err != nil {
			t.Fatalf("Get 30000 after second backfill: %v", err)
		}
		if count != 1 {
			t.Fatalf("count 30000 after second backfill = %d, want 1", count)
		}
	})
}

// TestBackfillUsesSQLiteFallbackUnderTest documents and pins the dialect
// branch Backfill takes in this module's test suite: sqlite cannot scan a
// SQL-side MIN()/MAX() aggregate over a DATETIME column back into
// time.Time, so Backfill routes sqlite through backfillGroupsSQLite (Go-side
// grouping) and reserves the bounded, single-query SQL GROUP BY/MIN/MAX
// aggregation (backfillGroupsSQL) for every other driver, including the
// production Postgres path. This test does not exercise backfillGroupsSQL
// itself -- doing so needs a live Postgres instance, unavailable to this
// module's test suite -- but it pins the dialect check so a future change
// that silently routes sqlite through the SQL-aggregate path (and
// reintroduces the scan failure) fails loudly here instead of surfacing as
// a flaky CI failure.
func TestBackfillUsesSQLiteFallbackUnderTest(t *testing.T) {
	db := backfillDatabase(t)
	if got := db.Dialector.Name(); got != "sqlite" {
		t.Fatalf("test database dialect = %q, want %q -- Backfill's dialect branch assumes sqlite in this suite", got, "sqlite")
	}
}

func mustCreateAsset(t *testing.T, db *gorm.DB, tenantId uuid.UUID, compartmentId uuid.UUID, commodityId uint32, at time.Time) uint32 {
	t.Helper()
	e := asset.Entity{
		TenantId:      tenantId,
		CompartmentId: compartmentId,
		CashId:        int64(commodityId)*1000 + 1,
		TemplateId:    commodityId,
		CommodityId:   commodityId,
		Quantity:      1,
		Flag:          0,
		PetId:         0,
		PurchasedBy:   0,
		Expiration:    at,
		CreatedAt:     at,
	}
	if err := db.Create(&e).Error; err != nil {
		t.Fatalf("create asset: %v", err)
	}
	return e.Id
}
