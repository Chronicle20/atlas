package handler

import (
	"atlas-channel/drop"
	"testing"
)

func mesoDrop(t *testing.T, id uint32, meso uint32) drop.Model {
	t.Helper()
	return drop.NewBuilder().SetId(id).SetMeso(meso).MustBuild()
}

func itemDrop(t *testing.T, id uint32) drop.Model {
	t.Helper()
	return drop.NewBuilder().SetId(id).SetItem(2000000, 1).MustBuild()
}

func TestValidateMesoExplosion(t *testing.T) {
	fieldDrops := map[uint32]drop.Model{
		11: mesoDrop(t, 11, 500),
		22: mesoDrop(t, 22, 120),
		33: itemDrop(t, 33),
	}

	tests := []struct {
		name      string
		dropIds   []uint32
		maxCount  uint32
		wantOk    bool
		wantBadId uint32
	}{
		{name: "happy path", dropIds: []uint32{11, 22}, maxCount: 10, wantOk: true},
		{name: "empty list is legal", dropIds: nil, maxCount: 10, wantOk: true},
		{name: "over skill max", dropIds: []uint32{11, 22}, maxCount: 1, wantOk: false, wantBadId: 0},
		{name: "duplicate id", dropIds: []uint32{11, 11}, maxCount: 10, wantOk: false, wantBadId: 11},
		{name: "unknown drop", dropIds: []uint32{11, 99}, maxCount: 10, wantOk: false, wantBadId: 99},
		{name: "non-meso drop", dropIds: []uint32{11, 33}, maxCount: 10, wantOk: false, wantBadId: 33},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			badId, ok := validateMesoExplosion(tc.dropIds, fieldDrops, tc.maxCount)
			if ok != tc.wantOk {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOk)
			}
			if !ok && badId != tc.wantBadId {
				t.Errorf("offending drop id = %d, want %d", badId, tc.wantBadId)
			}
		})
	}
}
