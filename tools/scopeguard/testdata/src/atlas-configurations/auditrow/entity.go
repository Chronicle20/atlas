package auditrow

// Entity is the fix-round-1 smuggle probe (task-232 Task 19): a
// control-plane entity with no Environment, no TenantId, and — critically —
// no uniquely-constrained natural key. It is exactly the shape the
// reviewer used to prove an allowlist-only gate was smuggleable: an
// audit/history row (RecordId/Actor/Note) that is plainly NOT "the row IS
// the environment," structurally indistinguishable from
// atlas-configurations/thing except for having an allowlist.txt entry.
// TestAnalyzerAllowlisted grants it one anyway and asserts it is STILL
// flagged — proving hasUniqueNaturalKey, not the allowlist line, is what
// gates the exception.
type Entity struct { // want `control-plane entity without Environment`
	Id       uint32
	RecordId string
	Actor    string
	Note     string
}
