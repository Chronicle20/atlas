package itementity

// The `Entity`-suffixed convention — mirrors atlas-trades/escrow's
// ItemEntity/MesoEntity. Has TenantId: passes.
type ItemEntity struct {
	Id       uint32
	TenantId uint32
	Name     string
}
