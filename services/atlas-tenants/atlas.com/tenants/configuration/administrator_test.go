package configuration_test

import (
	"atlas-tenants/configuration"
	"atlas-tenants/test"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// configTestDB reuses the service's shared in-memory sqlite helper
// (services/atlas-tenants/atlas.com/tenants/test/database.go), which
// already migrates tenant.Entity + configuration.Entity. The append and
// count administrators scope by an explicit tenant_id predicate, so no
// tenant GORM callback registration is needed here.
func configTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := test.SetupTestDB(t)
	t.Cleanup(func() { test.CleanupTestDB(db) })
	return db
}

func entry(id string) map[string]interface{} {
	return map[string]interface{}{
		"id":         id,
		"type":       "routes",
		"attributes": map[string]interface{}{"name": id},
	}
}

func TestAppendConfigurationEntries_CreatesThenAppends(t *testing.T) {
	db := configTestDB(t)
	tid := uuid.New()

	if err := configuration.AppendConfigurationEntries(db, tid, "routes", []map[string]interface{}{entry("a")}); err != nil {
		t.Fatalf("first append: %v", err)
	}
	if err := configuration.AppendConfigurationEntries(db, tid, "routes", []map[string]interface{}{entry("b")}); err != nil {
		t.Fatalf("second append: %v", err)
	}

	count, updatedAt, err := configuration.CountConfigurationEntries(db, tid, "routes")
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 2 {
		t.Fatalf("count = %d, want 2", count)
	}
	if updatedAt == nil {
		t.Fatal("updatedAt = nil, want the row's updated_at")
	}

	all, err := configuration.GetAllRoutesProvider(tid)(db)()
	if err != nil {
		t.Fatalf("GetAllRoutesProvider: %v", err)
	}
	if len(all) != 2 || all[0]["id"] != "a" || all[1]["id"] != "b" {
		t.Fatalf("entries = %+v, want [a b] in insertion order", all)
	}
}

func TestCountConfigurationEntries_NoRowIsZero(t *testing.T) {
	db := configTestDB(t)
	count, updatedAt, err := configuration.CountConfigurationEntries(db, uuid.New(), "routes")
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("count = %d, want 0", count)
	}
	if updatedAt != nil {
		t.Fatalf("updatedAt = %v, want nil", updatedAt)
	}
}

// The most severe possible regression for this task is a cross-tenant
// leak, so it gets a dedicated test at the administrator layer.
func TestAppendConfigurationEntries_IsTenantScoped(t *testing.T) {
	db := configTestDB(t)
	a, b := uuid.New(), uuid.New()

	if err := configuration.AppendConfigurationEntries(db, a, "routes", []map[string]interface{}{entry("only-a")}); err != nil {
		t.Fatalf("append for A: %v", err)
	}

	countB, _, err := configuration.CountConfigurationEntries(db, b, "routes")
	if err != nil {
		t.Fatalf("count for B: %v", err)
	}
	if countB != 0 {
		t.Fatalf("tenant B count = %d, want 0 — tenant A's seed leaked", countB)
	}

	if _, err := configuration.DeleteConfigurationByResourceName(db, b, "routes"); err != nil {
		t.Fatalf("delete for B: %v", err)
	}
	countA, _, err := configuration.CountConfigurationEntries(db, a, "routes")
	if err != nil {
		t.Fatalf("count for A: %v", err)
	}
	if countA != 1 {
		t.Fatalf("tenant A count = %d, want 1 — tenant B's delete wiped A", countA)
	}
}
