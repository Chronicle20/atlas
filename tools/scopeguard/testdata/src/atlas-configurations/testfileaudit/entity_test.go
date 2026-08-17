package testfileaudit

// Entity is the fix-round-2 smuggle probe (task-232 Task 19): the same
// audit-row shape as atlas-configurations/auditrow/entity.go (no scoping
// field, no uniquely-constrained natural key), but declared in a _test.go
// file instead of entity.go. It proves that entityAllowlistKey deriving
// its key from the declaring file's own base name — needed to let
// environments/processor_test.go's testEntity (a real, legitimate
// SQLite-compatible copy of environments.Entity) carry its own allowlist
// entry — does not create a blanket exemption for anything declared in a
// _test.go file. TestAnalyzerAllowlisted grants this an allowlist entry
// exactly like auditrow's, and asserts it is STILL flagged, because it
// fails hasUniqueNaturalKey regardless of which file declares it.
type Entity struct { // want `control-plane entity without Environment`
	Id       uint32
	RecordId string
	Actor    string
	Note     string
}
