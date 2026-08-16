package tenant_test

import (
	"atlas-tenants/scope"
	"atlas-tenants/tenant"
	"atlas-tenants/test"
	"context"
	"errors"
	"testing"

	logtest "github.com/sirupsen/logrus/hooks/test"
	"gorm.io/gorm"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// seedTenantWithEnvironment creates a tenant owned by environment directly
// at the entity layer, bypassing ProcessorImpl.Create (and its Kafka
// producer, unavailable in this unit test) - matches the existing
// testProcessor.create() / TestGetTenantsPaginates seeding precedent in
// this package.
func seedTenantWithEnvironment(t *testing.T, db *gorm.DB, environment string) tenant.Model {
	t.Helper()
	m, err := tenant.NewModelBuilder().
		SetName("Test Tenant " + environment).
		SetRegion("GMS").
		SetMajorVersion(83).
		SetMinorVersion(1).
		SetEnvironment(environment).
		Build()
	if err != nil {
		t.Fatalf("seedTenantWithEnvironment: Build() unexpected error: %v", err)
	}
	if err := tenant.CreateTenant(db, tenant.FromModel(m)); err != nil {
		t.Fatalf("seedTenantWithEnvironment: CreateTenant: %v", err)
	}
	return m
}

func TestGetAllReturnsOnlyTheCallersEnvironmentsTenants(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	seedTenantWithEnvironment(t, db, "main")
	seedTenantWithEnvironment(t, db, "pr-123")

	logger, _ := logtest.NewNullLogger()
	ctx := env.WithContext(context.Background(), env.Id("pr-123"))
	processor := tenant.NewProcessor(logger, ctx, db)
	paged, err := processor.AllProvider(model.Page{Number: 1, Size: 250})()
	if err != nil {
		t.Fatalf("AllProvider: %v", err)
	}
	if len(paged.Items) != 1 || paged.Items[0].Environment() != "pr-123" {
		t.Fatalf("got %d tenants (%v), want pr-123's only", len(paged.Items), paged.Items)
	}
}

func TestUpdateTenantRejectsCrossEnvironmentWrite(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	m := seedTenantWithEnvironment(t, db, "main")

	caller := env.WithContext(context.Background(), env.Id("pr-123"))
	e := tenant.FromModel(m)
	err := tenant.UpdateTenant(caller, db, e)
	if !errors.Is(err, scope.ErrCrossEnvironmentWrite) {
		t.Fatalf("UpdateTenant() error = %v, want scope.ErrCrossEnvironmentWrite", err)
	}
}

func TestDeleteTenantRejectsCrossEnvironmentWrite(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	m := seedTenantWithEnvironment(t, db, "main")

	caller := env.WithContext(context.Background(), env.Id("pr-123"))
	err := tenant.DeleteTenant(caller, db, m.Id())
	if !errors.Is(err, scope.ErrCrossEnvironmentWrite) {
		t.Fatalf("DeleteTenant() error = %v, want scope.ErrCrossEnvironmentWrite", err)
	}
}

func TestGetAllWithLegacyEnvironmentReturnsEverything(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	seedTenantWithEnvironment(t, db, "main")
	seedTenantWithEnvironment(t, db, "pr-123")

	logger, _ := logtest.NewNullLogger()
	processor := tenant.NewProcessor(logger, context.Background(), db)
	paged, err := processor.AllProvider(model.Page{Number: 1, Size: 250})()
	if err != nil {
		t.Fatalf("AllProvider: %v", err)
	}
	if len(paged.Items) != 2 {
		t.Fatalf("legacy GetAll returned %d, want 2 (FR-1.8)", len(paged.Items))
	}
}
