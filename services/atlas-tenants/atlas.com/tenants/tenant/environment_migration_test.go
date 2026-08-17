package tenant_test

import (
	"atlas-tenants/tenant"
	"atlas-tenants/test"
	"testing"

	"github.com/google/uuid"
)

func TestBackfillEnvironmentAssignsExistingRowsToTheBaseline(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	// A row written before the column existed has the zero value.
	db.Exec(`INSERT INTO tenants (id, name, region, major_version, minor_version, environment) VALUES (?, 'Test Tenant', 'GMS', 83, 1, '')`, uuid.New())

	if err := tenant.BackfillEnvironment(db, "main"); err != nil {
		t.Fatalf("BackfillEnvironment: %v", err)
	}

	var got string
	db.Raw(`SELECT environment FROM tenants LIMIT 1`).Scan(&got)
	if got != "main" {
		t.Fatalf("environment = %q, want \"main\"", got)
	}
}

func TestBackfillEnvironmentIsIdempotent(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	// A row that already carries a non-baseline environment must survive a
	// second (or first) backfill run untouched.
	id := uuid.New()
	db.Exec(`INSERT INTO tenants (id, name, region, major_version, minor_version, environment) VALUES (?, 'Test Tenant', 'GMS', 83, 1, 'pr-42')`, id)

	if err := tenant.BackfillEnvironment(db, "main"); err != nil {
		t.Fatalf("BackfillEnvironment (first run): %v", err)
	}
	if err := tenant.BackfillEnvironment(db, "main"); err != nil {
		t.Fatalf("BackfillEnvironment (second run): %v", err)
	}

	var got string
	db.Raw(`SELECT environment FROM tenants WHERE id = ?`, id).Scan(&got)
	if got != "pr-42" {
		t.Fatalf("environment = %q, want unchanged \"pr-42\"", got)
	}
}
