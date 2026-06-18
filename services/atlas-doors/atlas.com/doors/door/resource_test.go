package door

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

func TestTransform(t *testing.T) {
	f := field.NewBuilder(1, 2, 100000000).Build()
	m := NewBuilder().SetAreaDoorId(1_000_001).SetTownDoorId(1_000_002).
		SetOwnerCharacterId(42).SetTownMapId(104000000).SetSlot(2).
		SetTownPortalId(0x82).SetField(f).Build()
	rm, err := Transform(m)
	if err != nil || rm.GetID() != "1000001" || rm.OwnerCharacterId != 42 ||
		rm.TownPortalId != 0x82 || rm.MapId != 100000000 {
		t.Fatalf("transform wrong: %+v err=%v", rm, err)
	}
	if rm.GetName() != "doors" {
		t.Fatalf("resource name want doors got %s", rm.GetName())
	}
}
