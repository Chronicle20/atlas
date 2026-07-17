package gachapon_test

import (
	"atlas-reward-pools/gachapon"
	"atlas-reward-pools/test"
	"testing"
)

func TestUpdateGachaponNpcIds(t *testing.T) {
	processor, db, cleanup := test.CreateGachaponProcessor(t)
	defer cleanup()

	m, err := gachapon.NewBuilder(test.TestTenantId, "henesys").
		SetName("Henesys Gachapon").
		SetNpcIds([]uint32{9100100}).
		SetCommonWeight(70).
		SetUncommonWeight(25).
		SetRareWeight(5).
		Build()
	if err != nil {
		t.Fatalf("Failed to build gachapon model: %v", err)
	}
	if err = gachapon.CreateGachapon(db, m); err != nil {
		t.Fatalf("Failed to create gachapon: %v", err)
	}

	if err = processor.Update("henesys", "Henesys Gachapon", []uint32{9100100, 9100109}, 60, 30, 10); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, err := processor.GetById("henesys")
	if err != nil {
		t.Fatalf("GetById failed: %v", err)
	}
	if len(got.NpcIds()) != 2 || got.NpcIds()[0] != 9100100 || got.NpcIds()[1] != 9100109 {
		t.Errorf("npcIds not updated: %v", got.NpcIds())
	}
	if got.CommonWeight() != 60 {
		t.Errorf("weights not updated: common=%d", got.CommonWeight())
	}
	if got.Kind() != "gachapon" {
		t.Errorf("kind must be untouched: %q", got.Kind())
	}
}
