package mapimage

import "testing"

func TestSortObjRecsStable(t *testing.T) {
	recs := []objRec{
		{e: objEntry{idx: 0, z: 0, zM: 0}},
		{e: objEntry{idx: 1, z: 0, zM: 0}},
		{e: objEntry{idx: 2, z: -1, zM: 0}},
		{e: objEntry{idx: 3, z: 0, zM: 10}},
	}
	sortObjRecs(recs)
	// Expect: z=-1 first, then (z=0,zM=0) preserving idx order 0,1, then (z=0,zM=10).
	want := []int{2, 0, 1, 3}
	for i, r := range recs {
		if r.e.idx != want[i] {
			t.Errorf("pos %d idx=%d want %d", i, r.e.idx, want[i])
		}
	}
}

func TestSortTileRecsStable(t *testing.T) {
	s0 := &sprite{z: 0}
	sNeg := &sprite{z: -3}
	recs := []tileRec{
		{e: tileEntry{idx: 0, zM: 0}, s: s0},
		{e: tileEntry{idx: 1, zM: 0}, s: s0},
		{e: tileEntry{idx: 2, zM: 0}, s: sNeg},
		{e: tileEntry{idx: 3, zM: 5}, s: s0},
	}
	sortTileRecs(recs)
	want := []int{2, 0, 1, 3}
	for i, r := range recs {
		if r.e.idx != want[i] {
			t.Errorf("pos %d idx=%d want %d", i, r.e.idx, want[i])
		}
	}
}
