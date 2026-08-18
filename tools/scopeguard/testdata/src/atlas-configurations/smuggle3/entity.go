package smuggle3

// Entity is the fix-round-4 pin for the hasUniqueNaturalKey condition: a
// control-plane entity that declares the ScopingDimension marker AND has an
// allowlist entry, but NO uniquely-constrained non-surrogate field. Nothing
// in the committed suite before this fixture would notice if a future edit
// dropped `&& hasUniqueNaturalKey(st)` from checkEntity's control-plane
// branch — smuggle2 pins the marker condition and testfileaudit pins the
// _test.go file-key derivation, but neither exercises "marker present,
// allowlist present, unique key ABSENT." This one does: it must still be
// flagged, because a marker plus an allowlist line is not enough on its
// own — the entity must also structurally own its identity via a real DB
// uniqueness constraint.
type Entity struct { // want `control-plane entity without Environment`
	Id       uint32
	RecordId string
	Actor    string
	Note     string
}

func (Entity) ScopingDimension() {}
