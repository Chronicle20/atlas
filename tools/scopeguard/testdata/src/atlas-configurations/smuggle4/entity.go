package smuggle4

// Entity is the fix-round-4 pin for the EntityAllowlist condition: a
// control-plane entity that declares the ScopingDimension marker AND has a
// uniquely-constrained non-surrogate field (Name), but is deliberately
// given NO allowlist.txt entry. Nothing in the committed suite before this
// fixture would notice if a future edit dropped the `EntityAllowlist[key]`
// lookup from checkEntity's control-plane branch — a marker plus a unique
// key is not enough on its own; the exemption must also be recorded in
// allowlist.txt, the one place a reviewer sees every excused entity listed
// in one file. This is deliberately run through TestAnalyzer (the real,
// checked-in allowlist.txt, which carries no entry for this package), not
// TestAnalyzerAllowlisted's fixture map, since the whole point is that no
// allowlist entry exists for it anywhere.
type Entity struct { // want `control-plane entity without Environment`
	Id   uint32
	Name string `gorm:"uniqueIndex"`
}

func (Entity) ScopingDimension() {}
