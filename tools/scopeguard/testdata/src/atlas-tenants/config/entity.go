package config

// Entity has a TenantId even though it lives in a control-plane service —
// it is per-tenant data (mirrors atlas-tenants' real configuration.Entity),
// scoped by the fleet-wide GORM callback like any other data-plane entity.
// No diagnostic expected.
type Entity struct {
	Id       uint32
	TenantId uint32
	Data     string
}
