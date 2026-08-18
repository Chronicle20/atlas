package family

import (
	"context"
	"testing"

	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// newMigrationTestDB starts a disposable Postgres container and returns a
// connected *gorm.DB, mirroring the container setup in atlas-character's
// pending_change/entity_test.go.
//
// Migration()'s Postgres branch cannot be exercised against SQLite: the
// dialect check skips every ALTER TABLE ... ADD CONSTRAINT there, so the
// constraint SQL is never parsed and a malformed expression passes unnoticed.
// That is exactly how the array_length() regression below reached a live
// cluster.
func newMigrationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	ctx := context.Background()
	pg, err := tcpostgres.Run(ctx, "postgres:16-alpine", tcpostgres.BasicWaitStrategies())
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	t.Cleanup(func() { _ = pg.Terminate(ctx) })

	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open gorm: %v", err)
	}
	return db
}

// TestMigrationSucceedsOnPostgres pins the startup path. Migration() runs
// before the service serves anything, so any statement it issues that Postgres
// rejects is fatal: the process logs "Migrating schema" at level fatal and
// exits, and the pod crash-loops.
//
// The regression this guards: JuniorIds is persisted with `serializer:json`,
// making junior_ids a TEXT column holding a JSON array, but check_junior_count
// was written with array_length() — a Postgres ARRAY function. It failed with
// "function array_length(text, integer) does not exist" (SQLSTATE 42883) the
// first time the service was actually deployed.
func TestMigrationSucceedsOnPostgres(t *testing.T) {
	db := newMigrationTestDB(t)

	if err := Migration(db); err != nil {
		t.Fatalf("Migration must succeed against Postgres, got: %v", err)
	}

	// The serializer:json column really is text — the premise the constraint
	// expression has to be written against.
	var colType string
	if err := db.Raw(`SELECT data_type FROM information_schema.columns
		WHERE table_name = 'family_members' AND column_name = 'junior_ids'`).Scan(&colType).Error; err != nil {
		t.Fatalf("introspect junior_ids: %v", err)
	}
	if colType != "text" {
		t.Fatalf("junior_ids column type = %q, want \"text\" — the check_junior_count expression assumes JSON-in-text", colType)
	}

	var n int64
	if err := db.Raw(`SELECT count(*) FROM pg_constraint WHERE conname = 'check_junior_count'`).Scan(&n).Error; err != nil {
		t.Fatalf("count constraint: %v", err)
	}
	if n != 1 {
		t.Fatalf("check_junior_count constraint count = %d, want 1", n)
	}

	// Migration is re-run on every start, so it must be idempotent.
	if err := Migration(db); err != nil {
		t.Fatalf("second Migration run must be a no-op, got: %v", err)
	}
}

// TestJuniorCountConstraintEnforced proves the constraint still does its job
// after the rewrite — a green migration that silently enforces nothing would
// pass the test above.
func TestJuniorCountConstraintEnforced(t *testing.T) {
	db := newMigrationTestDB(t)
	if err := Migration(db); err != nil {
		t.Fatalf("Migration: %v", err)
	}

	insert := func(characterId int, juniorIds interface{}) error {
		return db.Exec(`INSERT INTO family_members
			(character_id, tenant_id, junior_ids, rep, daily_rep, level, world, created_at, updated_at)
			VALUES (?, gen_random_uuid(), ?, 0, 0, 1, 0, now(), now())`, characterId, juniorIds).Error
	}

	// A nil slice serializes to SQL NULL, and an empty slice to "[]"; both are
	// legitimate states for a member with no juniors.
	if err := insert(1, nil); err != nil {
		t.Errorf("NULL junior_ids must be accepted: %v", err)
	}
	if err := insert(2, "[]"); err != nil {
		t.Errorf("empty junior_ids must be accepted: %v", err)
	}
	if err := insert(3, "[10,11]"); err != nil {
		t.Errorf("two juniors must be accepted: %v", err)
	}
	// Two is the cap (ValidateJuniorIds enforces the same rule in the domain
	// layer); the constraint is the backstop.
	if err := insert(4, "[10,11,12]"); err == nil {
		t.Error("three juniors must be rejected by check_junior_count, but the insert succeeded")
	}
}
