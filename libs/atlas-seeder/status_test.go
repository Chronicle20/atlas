package seeder

import (
	"context"
	"testing"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func TestStatus_NeverSeededReturnsNilTenantFields(t *testing.T) {
	db := openTestDB(t)
	src := NewFilesystemCatalogSource("X_NO_ENV", goodFixtureRoot(t))
	g := Group{
		Name:       "widgets-group",
		URLPrefix:  "/widgets",
		Subdomains: []SubdomainAny{AdaptSubdomain[widgetAttrs, widgetRow](&widgetSubdomain{})},
	}
	ctx := tenant.WithContext(context.Background(), tenantGMS83(t))
	st, err := ReadStatus(ctx, db, src, g)
	if err != nil {
		t.Fatalf("ReadStatus: %v", err)
	}
	if st.CatalogRevision != "test-rev-abc123" {
		t.Fatalf("catalogRevision = %q", st.CatalogRevision)
	}
	if st.TenantSeededRevision != nil {
		t.Fatalf("TenantSeededRevision = %v, want nil", st.TenantSeededRevision)
	}
	if _, ok := st.Subdomains["widgets"]; !ok {
		t.Fatalf("subdomain entry missing")
	}
}

func TestStatus_AfterSeedPopulatesTenantRevision(t *testing.T) {
	db := openTestDB(t)
	t.Cleanup(ResetMetricsForTest)
	src := NewFilesystemCatalogSource("X_NO_ENV", goodFixtureRoot(t))
	g := Group{
		Name:       "widgets-group",
		URLPrefix:  "/widgets",
		Subdomains: []SubdomainAny{AdaptSubdomain[widgetAttrs, widgetRow](&widgetSubdomain{})},
	}
	ctx := tenant.WithContext(context.Background(), tenantGMS83(t))
	if _, err := Seed(ctx, db, src, g); err != nil {
		t.Fatalf("Seed: %v", err)
	}
	st, err := ReadStatus(ctx, db, src, g)
	if err != nil {
		t.Fatalf("ReadStatus: %v", err)
	}
	if st.TenantSeededRevision == nil || *st.TenantSeededRevision != "test-rev-abc123" {
		t.Fatalf("TenantSeededRevision = %v", st.TenantSeededRevision)
	}
	if st.TenantSeededAt == nil {
		t.Fatalf("TenantSeededAt = nil")
	}
}
