package smuggle2

// Entity is the fix-round-2 re-reviewer's smuggle probe (task-232 Task 19,
// fix round 3): a control-plane row with no Environment and no TenantId,
// but WITH a uniquely-constrained non-surrogate field (RequestId, an
// idempotency key) — the shape hasUniqueNaturalKey alone could not
// distinguish from a genuine "this row IS the scoping dimension" entity.
// An idempotency-keyed job/audit record is emphatically not the top-level
// enumeration of anything; it merely happens to also carry a real DB
// uniqueness constraint, on an unrelated column, for an unrelated reason.
//
// TestAnalyzerAllowlisted grants this an allowlist entry exactly like
// envfixture's, and it satisfies hasUniqueNaturalKey (RequestId is
// non-surrogate, uniqueIndex-tagged) — but it deliberately declares NO
// ScopingDimension marker method. It must still be flagged, proving the
// marker (not the allowlist line, not the shape check) is what closes the
// round-2 hole.
type Entity struct { // want `control-plane entity without Environment`
	Id        uint32
	RequestId string `gorm:"uniqueIndex"`
	Actor     string
	Note      string
}
