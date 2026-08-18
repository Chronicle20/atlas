package registry

// Entity is the tenant registry row itself — no TenantId is possible (the
// row IS the tenant), scoped instead by Environment. No diagnostic
// expected: this is the passing control-plane shape (mirrors
// atlas-tenants' real tenant.Entity).
type Entity struct {
	Id          uint32
	Name        string
	Environment string
}
