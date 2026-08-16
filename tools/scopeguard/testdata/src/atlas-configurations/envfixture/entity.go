package envfixture

// Entity is the control-plane "the row IS the scoping dimension" shape
// (task-232 Task 19, atlas-configurations/environments.Entity): no
// TenantId (not tenant data) and no Environment (the row IS the
// environment). Name carries the uniqueIndex tag the structural check
// (hasUniqueNaturalKey) requires — mirroring environments.Entity's real
// `Name string `gorm:"not null;uniqueIndex"“ column. It also declares the
// ScopingDimension marker method (fix round 3), mirroring the real
// entity's own marker. This exact key is present in the test allowlist
// fixture in analyzer_test.go's TestAnalyzerAllowlisted — no diagnostic
// expected, since all three conditions (allowlist, unique natural key,
// marker) hold.
type Entity struct {
	Id   uint32
	Name string `gorm:"uniqueIndex"`
}

func (Entity) ScopingDimension() {}
