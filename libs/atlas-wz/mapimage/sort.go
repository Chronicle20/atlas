package mapimage

import "sort"

// objRec and tileRec pair an entry with its resolved sprite for post-sort blitting.
type objRec struct {
	e objEntry
	s *sprite
}

type tileRec struct {
	e tileEntry
	s *sprite
}

// sortObjRecs sorts objs by (z, zM, insertion_index) ascending. Stable.
func sortObjRecs(recs []objRec) {
	sort.SliceStable(recs, func(i, j int) bool {
		if recs[i].e.z != recs[j].e.z {
			return recs[i].e.z < recs[j].e.z
		}
		if recs[i].e.zM != recs[j].e.zM {
			return recs[i].e.zM < recs[j].e.zM
		}
		return recs[i].e.idx < recs[j].e.idx
	})
}

// sortTileRecs sorts tiles by (sprite.z, entry.zM, insertion_index) ascending. Stable.
func sortTileRecs(recs []tileRec) {
	sort.SliceStable(recs, func(i, j int) bool {
		if recs[i].s.z != recs[j].s.z {
			return recs[i].s.z < recs[j].s.z
		}
		if recs[i].e.zM != recs[j].e.zM {
			return recs[i].e.zM < recs[j].e.zM
		}
		return recs[i].e.idx < recs[j].e.idx
	})
}
