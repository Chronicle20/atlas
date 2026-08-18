package widget

// Entity has no TenantId, but this exact key is present in the test
// allowlist fixture below (see analyzer_test.go) — no diagnostic expected.
type Entity struct {
	Id   uint32
	Name string
}
