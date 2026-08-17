package scoped

// Entity has a TenantId — scoped by the fleet-wide GORM callback,
// regardless of service. No diagnostic expected.
type Entity struct {
	Id       uint32
	TenantId uint32
	Name     string
}
