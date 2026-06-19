package door

import (
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

func TestBuilderAndGettersAndReslotImmutable(t *testing.T) {
	f := field.NewBuilder(1, 2, 100000000).Build()
	deploy := time.Unix(1000, 0)
	m := NewBuilder().
		SetAreaDoorId(1_000_001).
		SetTownDoorId(1_000_002).
		SetOwnerCharacterId(42).
		SetPartyId(7).
		SetSkillId(2311002).
		SetSkillLevel(10).
		SetField(f).
		SetTownMapId(104000000).
		SetSlot(0).
		SetTownPortalId(0x80).
		SetAreaX(50).SetAreaY(60).
		SetTownX(-12).SetTownY(34).
		SetDeployTime(deploy).
		SetExpiresAt(deploy.Add(2 * time.Minute)).
		Build()

	if m.AreaDoorId() != 1_000_001 || m.TownDoorId() != 1_000_002 {
		t.Fatalf("door ids wrong: %d/%d", m.AreaDoorId(), m.TownDoorId())
	}
	if m.PairId() != m.AreaDoorId() {
		t.Fatalf("pairId must equal areaDoorId, got %d", m.PairId())
	}
	if m.Field().MapId() != 100000000 {
		t.Fatalf("field map wrong: %d", m.Field().MapId())
	}

	// Reslot returns a NEW model; original unchanged.
	n := m.Reslot(3, 0x83, -99, 88)
	if m.Slot() != 0 || m.TownPortalId() != 0x80 || m.TownX() != -12 {
		t.Fatalf("original mutated by Reslot")
	}
	if n.Slot() != 3 || n.TownPortalId() != 0x83 || n.TownX() != -99 || n.TownY() != 88 {
		t.Fatalf("reslot did not apply: slot=%d portal=%d x=%d y=%d", n.Slot(), n.TownPortalId(), n.TownX(), n.TownY())
	}
	// Reslot preserves identity fields.
	if n.AreaDoorId() != m.AreaDoorId() || n.OwnerCharacterId() != m.OwnerCharacterId() {
		t.Fatalf("reslot changed identity fields")
	}
}
