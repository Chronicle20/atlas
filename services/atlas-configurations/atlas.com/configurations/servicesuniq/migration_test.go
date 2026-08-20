package servicesuniq

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	outboxlib "github.com/Chronicle20/atlas/libs/atlas-outbox"
)

// The production services.Entity declares postgres-only column
// defaults/types that SQLite's CREATE TABLE syntax cannot express. This
// test follows the same shadow-entity pattern as
// environmentcol/migration_test.go rather than AutoMigrate-ing the
// production struct directly. outbox_entries has no such incompatibility,
// so it is migrated with the library's own outboxlib.Migration(db) instead
// of a local shadow struct — the fleet convention, e.g.
// services/atlas-tenants/atlas.com/tenants/configuration/rankings_handler_test.go
// and services/atlas-guilds/.../thread/processor_outbox_test.go.

type testServiceEntity struct {
	Id          uuid.UUID       `gorm:"type:text;primaryKey"`
	Type        string          `gorm:"type:text;not null"`
	Data        json.RawMessage `gorm:"type:text;not null"`
	Environment string          `gorm:"not null;default:''"`
}

func (testServiceEntity) TableName() string { return "services" }

type testServiceHistoryEntity struct {
	Id          uuid.UUID       `gorm:"type:text;primaryKey"`
	ServiceId   uuid.UUID       `gorm:"type:text"`
	Type        string          `gorm:"type:text;not null"`
	Data        json.RawMessage `gorm:"type:text;not null"`
	CreatedAt   time.Time       `gorm:"not null"`
	Environment string          `gorm:"not null;default:''"`
}

func (testServiceHistoryEntity) TableName() string { return "service_history" }

// testDatabase creates an in-memory SQLite database migrated with the
// SQLite-compatible shadow entities servicesuniq touches, plus the real
// outbox_entries table via outboxlib.Migration.
func testDatabase(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get underlying sql.DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)

	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	if err := db.AutoMigrate(&testServiceEntity{}, &testServiceHistoryEntity{}); err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	if err := outboxlib.Migration(db); err != nil {
		t.Fatalf("Failed to migrate outbox_entries: %v", err)
	}

	return db
}

func rowCount(t *testing.T, db *gorm.DB, table string) int {
	t.Helper()
	var count int
	if err := db.Raw("SELECT COUNT(*) FROM " + table).Scan(&count).Error; err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return count
}

func TestDedupeKeepsTheDerivedIdRow(t *testing.T) {
	db := testDatabase(t)

	db.Exec(`INSERT INTO services (id, type, data, environment) VALUES (?, 'login-service', '{}', 'pr-1411')`, "6439ca9c-d28d-5db9-821b-8dd93d318a25")
	db.Exec(`INSERT INTO services (id, type, data, environment) VALUES (?, 'login-service', '{}', 'pr-1411')`, "11111111-1111-1111-1111-111111111111")
	db.Exec(`INSERT INTO services (id, type, data, environment) VALUES (?, 'login-service', '{}', 'pr-1411')`, "22222222-2222-2222-2222-222222222222")

	if err := Migration(db); err != nil {
		t.Fatalf("Migration: %v", err)
	}

	if got := rowCount(t, db, "services"); got != 1 {
		t.Fatalf("services row count = %d, want 1", got)
	}

	var got string
	db.Raw(`SELECT id FROM services LIMIT 1`).Scan(&got)
	if got != "6439ca9c-d28d-5db9-821b-8dd93d318a25" {
		t.Fatalf("surviving id = %q, want the derived id", got)
	}
}

func TestDedupeKeepsTheNewestWhenNoDerivedIdMatches(t *testing.T) {
	db := testDatabase(t)

	db.Exec(`INSERT INTO services (id, type, data, environment) VALUES (?, 'drops-service', '{}', 'pr-1411')`, "11111111-1111-1111-1111-111111111111")
	db.Exec(`INSERT INTO services (id, type, data, environment) VALUES (?, 'drops-service', '{}', 'pr-1411')`, "22222222-2222-2222-2222-222222222222")

	db.Exec(`INSERT INTO service_history (id, service_id, type, data, created_at, environment) VALUES (?, ?, 'drops-service', '{}', ?, 'pr-1411')`,
		uuid.New(), "11111111-1111-1111-1111-111111111111", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	db.Exec(`INSERT INTO service_history (id, service_id, type, data, created_at, environment) VALUES (?, ?, 'drops-service', '{}', ?, 'pr-1411')`,
		uuid.New(), "22222222-2222-2222-2222-222222222222", time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC))

	if err := Migration(db); err != nil {
		t.Fatalf("Migration: %v", err)
	}

	if got := rowCount(t, db, "services"); got != 1 {
		t.Fatalf("services row count = %d, want 1", got)
	}

	var got string
	db.Raw(`SELECT id FROM services LIMIT 1`).Scan(&got)
	if got != "22222222-2222-2222-2222-222222222222" {
		t.Fatalf("surviving id = %q, want the newest-history id", got)
	}
}

func TestDedupeFallsBackToTheLowestIdWithNoHistory(t *testing.T) {
	db := testDatabase(t)

	db.Exec(`INSERT INTO services (id, type, data, environment) VALUES (?, 'world-service', '{}', 'pr-1411')`, "22222222-2222-2222-2222-222222222222")
	db.Exec(`INSERT INTO services (id, type, data, environment) VALUES (?, 'world-service', '{}', 'pr-1411')`, "11111111-1111-1111-1111-111111111111")

	if err := Migration(db); err != nil {
		t.Fatalf("Migration: %v", err)
	}

	if got := rowCount(t, db, "services"); got != 1 {
		t.Fatalf("services row count = %d, want 1", got)
	}

	var got string
	db.Raw(`SELECT id FROM services LIMIT 1`).Scan(&got)
	if got != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("surviving id = %q, want the lowest id", got)
	}
}

func TestDedupeEnqueuesATombstoneForEveryDeletedRow(t *testing.T) {
	t.Setenv("EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS", "test.svc.topic")

	db := testDatabase(t)

	db.Exec(`INSERT INTO services (id, type, data, environment) VALUES (?, 'login-service', '{}', 'pr-1411')`, "6439ca9c-d28d-5db9-821b-8dd93d318a25")
	db.Exec(`INSERT INTO services (id, type, data, environment) VALUES (?, 'login-service', '{}', 'pr-1411')`, "11111111-1111-1111-1111-111111111111")
	db.Exec(`INSERT INTO services (id, type, data, environment) VALUES (?, 'login-service', '{}', 'pr-1411')`, "22222222-2222-2222-2222-222222222222")

	if err := Migration(db); err != nil {
		t.Fatalf("Migration: %v", err)
	}

	if got := rowCount(t, db, "outbox_entries"); got != 2 {
		t.Fatalf("outbox_entries row count = %d, want 2", got)
	}

	var rows []outboxlib.Entity
	if err := db.Find(&rows).Error; err != nil {
		t.Fatalf("query outbox_entries: %v", err)
	}

	wantKeys := map[string]bool{
		"service:11111111-1111-1111-1111-111111111111": false,
		"service:22222222-2222-2222-2222-222222222222": false,
	}
	for _, r := range rows {
		if r.Topic != "test.svc.topic" {
			t.Fatalf("topic = %q, want %q", r.Topic, "test.svc.topic")
		}
		key := string(r.MessageKey)
		if _, ok := wantKeys[key]; !ok {
			t.Fatalf("unexpected message_key %q", key)
		}
		wantKeys[key] = true
		if len(r.MessageValue) != 0 {
			t.Fatalf("message_value = %q, want empty (tombstone)", r.MessageValue)
		}
	}
	for key, seen := range wantKeys {
		if !seen {
			t.Fatalf("missing tombstone for %q", key)
		}
	}
}

func TestDedupeLeavesOtherEnvironmentsAlone(t *testing.T) {
	db := testDatabase(t)

	db.Exec(`INSERT INTO services (id, type, data, environment) VALUES (?, 'login-service', '{}', 'main')`, uuid.New())
	db.Exec(`INSERT INTO services (id, type, data, environment) VALUES (?, 'login-service', '{}', 'pr-1411')`, uuid.New())

	if err := Migration(db); err != nil {
		t.Fatalf("Migration: %v", err)
	}

	if got := rowCount(t, db, "services"); got != 2 {
		t.Fatalf("services row count = %d, want 2", got)
	}
}

func TestMigrationIsIdempotent(t *testing.T) {
	db := testDatabase(t)

	db.Exec(`INSERT INTO services (id, type, data, environment) VALUES (?, 'login-service', '{}', 'pr-1411')`, "6439ca9c-d28d-5db9-821b-8dd93d318a25")
	db.Exec(`INSERT INTO services (id, type, data, environment) VALUES (?, 'login-service', '{}', 'pr-1411')`, "11111111-1111-1111-1111-111111111111")
	db.Exec(`INSERT INTO services (id, type, data, environment) VALUES (?, 'login-service', '{}', 'pr-1411')`, "22222222-2222-2222-2222-222222222222")

	if err := Migration(db); err != nil {
		t.Fatalf("Migration (first run): %v", err)
	}

	rowsBefore := rowCount(t, db, "services")
	outboxBefore := rowCount(t, db, "outbox_entries")

	if err := Migration(db); err != nil {
		t.Fatalf("Migration (second run): %v", err)
	}

	if got := rowCount(t, db, "services"); got != rowsBefore {
		t.Fatalf("services row count changed on second run: %d -> %d", rowsBefore, got)
	}
	if got := rowCount(t, db, "outbox_entries"); got != outboxBefore {
		t.Fatalf("outbox_entries row count changed on second run: %d -> %d", outboxBefore, got)
	}
}

func TestMigrationCreatesTheUniqueIndex(t *testing.T) {
	db := testDatabase(t)

	db.Exec(`INSERT INTO services (id, type, data, environment) VALUES (?, 'login-service', '{}', 'main')`, uuid.New())
	db.Exec(`INSERT INTO services (id, type, data, environment) VALUES (?, 'login-service', '{}', 'pr-1411')`, uuid.New())

	if err := Migration(db); err != nil {
		t.Fatalf("Migration: %v", err)
	}

	err := db.Exec(`INSERT INTO services (id, type, data, environment) VALUES (?, 'login-service', '{}', 'pr-1411')`, uuid.New()).Error
	if err == nil {
		t.Fatal("expected duplicate insert to fail against the unique index, got nil error")
	}
}

func TestPreflightNamesDuplicateGroups(t *testing.T) {
	db := testDatabase(t)

	db.Exec(`INSERT INTO services (id, type, data, environment) VALUES (?, 'login-service', '{}', 'pr-1411')`, "6439ca9c-d28d-5db9-821b-8dd93d318a25")
	db.Exec(`INSERT INTO services (id, type, data, environment) VALUES (?, 'login-service', '{}', 'pr-1411')`, "11111111-1111-1111-1111-111111111111")
	db.Exec(`INSERT INTO services (id, type, data, environment) VALUES (?, 'login-service', '{}', 'pr-1411')`, "22222222-2222-2222-2222-222222222222")

	before := rowCount(t, db, "services")

	groups, err := Preflight(db)
	if err != nil {
		t.Fatalf("Preflight: %v", err)
	}

	if len(groups) != 1 {
		t.Fatalf("groups = %d, want 1", len(groups))
	}
	want := DuplicateGroup{Type: "login-service", Environment: "pr-1411", Count: 3}
	if groups[0] != want {
		t.Fatalf("group = %+v, want %+v", groups[0], want)
	}

	if got := rowCount(t, db, "services"); got != before {
		t.Fatalf("Preflight mutated data: row count %d -> %d", before, got)
	}
}
