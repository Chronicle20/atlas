package monster

import "sort"

type interval struct {
	lo int
	hi int
}

// intervalSet is a union of possibly-overlapping level bands, ported from
// <cosmic> tools/IntervalBuilder.java. FR-6.2 is explicit that the level gate
// is a union and NOT a single [min-5, max+5] band: contributors at levels 30
// and 120 against a level-125 mob must admit a level-32 member and reject a
// level-70 one.
type intervalSet struct {
	ivs []interval
}

// add records one band. lo is clamped at 0 because callers do unsigned level
// arithmetic in int and a low-level mob can produce a negative lower bound.
// Merging is deferred to build() so add stays O(1).
func (s *intervalSet) add(lo, hi int) {
	if lo < 0 {
		lo = 0
	}
	if hi < lo {
		return
	}
	s.ivs = append(s.ivs, interval{lo: lo, hi: hi})
}

// build returns a copy with the bands sorted by lo and overlapping or
// adjacent bands merged.
func (s intervalSet) build() intervalSet {
	ivs := make([]interval, len(s.ivs))
	copy(ivs, s.ivs)

	sort.Slice(ivs, func(i, j int) bool {
		return ivs[i].lo < ivs[j].lo
	})

	merged := make([]interval, 0, len(ivs))
	for _, iv := range ivs {
		if len(merged) > 0 && iv.lo <= merged[len(merged)-1].hi+1 {
			last := &merged[len(merged)-1]
			if iv.hi > last.hi {
				last.hi = iv.hi
			}
			continue
		}
		merged = append(merged, iv)
	}

	return intervalSet{ivs: merged}
}

// contains reports whether v falls in any band. A linear scan is correct and
// clearer than a binary search at these sizes: one mob band plus at most six
// contributor bands.
func (s intervalSet) contains(v int) bool {
	for _, iv := range s.ivs {
		if v >= iv.lo && v <= iv.hi {
			return true
		}
	}
	return false
}
