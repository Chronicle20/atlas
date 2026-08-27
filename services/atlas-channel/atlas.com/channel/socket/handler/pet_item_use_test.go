package handler

import (
	"atlas-channel/pet"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
)

func TestEvaluateAutoPot(t *testing.T) {
	cases := []struct {
		name                     string
		characterHp              uint16
		petOwnerId               uint32
		petSlot                  int8
		recoversHP, recoversMP   bool
		hasHPSource, hasMPSource bool
		wantReason               string
		wantOk                   bool
	}{
		{"happy hp", 100, 1, 0, true, false, true, false, "", true},
		{"happy mp", 100, 1, 0, false, true, false, true, "", true},
		{"dual either hp source", 100, 1, 0, true, true, true, false, "", true},
		{"dual either mp source", 100, 1, 0, true, true, false, true, "", true},
		{"not owned", 100, 2, 0, true, false, true, false, "pet_not_owned", false},
		{"not spawned", 100, 1, -1, true, false, true, false, "pet_not_spawned", false},
		{"dead", 0, 1, 0, true, false, true, false, "character_dead", false},
		{"missing hp skill", 100, 1, 0, true, false, false, true, "missing_pet_skill", false},
		{"missing mp skill", 100, 1, 0, false, true, true, false, "missing_pet_skill", false},
		{"not a potion", 100, 1, 0, false, false, true, true, "not_consumable", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			reason, ok := evaluateAutoPot(1, c.characterHp, c.petOwnerId, c.petSlot, c.recoversHP, c.recoversMP, c.hasHPSource, c.hasMPSource)
			if ok != c.wantOk || reason != c.wantReason {
				t.Errorf("got (%q,%v), want (%q,%v)", reason, ok, c.wantReason, c.wantOk)
			}
		})
	}
}

// TestResolveSpawnedPet covers the gms_48 pet-resolution branch (design
// amendment): the v48 wire carries no petId, so the lookup in front of
// evaluateAutoPot resolves the character's spawned pet instead of a
// petId-bearing GetById. This is never a fallback from the petId-bearing
// path — resolveSpawnedPet is only ever called when the wire petId was
// absent (petId == 0), and the petId-bearing path never calls it.
func TestResolveSpawnedPet(t *testing.T) {
	spawned := pet.NewBuilder(1, 1234567890, 5000000, "Fluffy").SetOwnerID(1).SetSlot(0).MustBuild()
	despawned := pet.NewBuilder(2, 1234567891, 5000000, "Rex").SetOwnerID(1).SetSlot(-1).MustBuild()

	cases := []struct {
		name       string
		pets       []pet.Model
		wantId     uint32
		wantReason string
		wantOk     bool
	}{
		{"v48 no petId, spawned pet", []pet.Model{despawned, spawned}, 1, "", true},
		{"v48 no petId, no spawned pet", []pet.Model{despawned}, 0, "pet_not_found", false},
		{"v48 no petId, no pets at all", nil, 0, "pet_not_found", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			pm, reason, ok := resolveSpawnedPet(c.pets)
			if ok != c.wantOk || reason != c.wantReason {
				t.Errorf("got (%q,%v), want (%q,%v)", reason, ok, c.wantReason, c.wantOk)
			}
			if ok && pm.Id() != c.wantId {
				t.Errorf("pm.Id() = %d, want %d", pm.Id(), c.wantId)
			}
		})
	}
}

func TestPetAbilityPositions(t *testing.T) {
	cases := []struct {
		petSlot int8
		in      slot.Position
		want    bool
	}{
		{0, -24, true},  // petHP, always honored
		{0, -25, true},  // petMP, always honored
		{0, -21, true},  // pet-0 ability range
		{0, -46, true},  // petItemIgnore (pet 0)
		{0, -31, false}, // pet-1 range does not apply to pet 0
		{1, -31, true},
		{1, -24, true}, // shared slots apply to every pet index
		{1, -21, false},
		{2, -39, true},
		{2, -48, true},
		{2, -46, false},
		{0, -7, false}, // ordinary equip slot never matches
	}
	for _, c := range cases {
		got := petAbilityPositions(c.petSlot)[c.in]
		if got != c.want {
			t.Errorf("petAbilityPositions(%d)[%d] = %v, want %v", c.petSlot, c.in, got, c.want)
		}
	}
}

func TestNormalizeWornPosition(t *testing.T) {
	// Worn cash equips are stored at position-100 in the raw equip compartment
	// (see character.Model.SetInventory); pet equips are cash items.
	if got := normalizeWornPosition(-124); got != -24 {
		t.Errorf("normalizeWornPosition(-124) = %d, want -24", got)
	}
	if got := normalizeWornPosition(-24); got != -24 {
		t.Errorf("normalizeWornPosition(-24) = %d, want -24", got)
	}
	if got := normalizeWornPosition(5); got != 5 {
		t.Errorf("normalizeWornPosition(5) = %d, want 5", got)
	}
}

func TestMatchPetAbilityEquips(t *testing.T) {
	worn := []wornEquip{
		{position: -124, abilities: []string{"consumeHP"}},
		{position: -31, abilities: []string{"consumeMP"}},
	}
	// pet 0: -124 normalizes to -24 (shared) -> HP yes; -31 is pet-1 range -> MP no.
	hasHP, hasMP, sawData := matchPetAbilityEquips(worn, 0)
	if !hasHP || hasMP || !sawData {
		t.Errorf("pet 0: got (%v,%v,%v), want (true,false,true)", hasHP, hasMP, sawData)
	}
	// pet 1: shared -24 HP yes; -31 in range -> MP yes.
	hasHP, hasMP, _ = matchPetAbilityEquips(worn, 1)
	if !hasHP || !hasMP {
		t.Errorf("pet 1: got (%v,%v), want (true,true)", hasHP, hasMP)
	}
	// no attribute data at all -> sawData false (drives equip_data_missing).
	_, _, sawData = matchPetAbilityEquips([]wornEquip{{position: -124, abilities: nil}}, 0)
	if sawData {
		t.Error("sawData = true with no ability data, want false")
	}
}

// TestClassifyPetIdInput pins the version-gated pet-resolution decision. The
// wire petId is present on GMS v61+ and all JMS, absent on gms_48. Deciding on
// `petId != 0` alone cannot tell "this version has no petId field" from "this
// version has one and the client sent literal 0" — the latter is malformed or
// forged and must be rejected, not quietly routed into the spawned-pet branch
// (the FR-1 invariant the handler comment states: never fall back from one
// resolution path to the other).
func TestClassifyPetIdInput(t *testing.T) {
	cases := []struct {
		name         string
		hasWirePetId bool
		petId        uint64
		wantUsePetId bool
		wantReason   string
		wantOk       bool
	}{
		{"v48 no wire field", false, 0, false, "", true},
		{"v48 ignores a stray non-zero", false, 99, false, "", true},
		{"v61+ real petId", true, 987654321, true, "", true},
		{"v61+ zero petId is rejected", true, 0, false, "pet_not_found", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			usePetId, reason, ok := classifyPetIdInput(c.hasWirePetId, c.petId)
			if usePetId != c.wantUsePetId || reason != c.wantReason || ok != c.wantOk {
				t.Errorf("got (%v,%q,%v), want (%v,%q,%v)", usePetId, reason, ok, c.wantUsePetId, c.wantReason, c.wantOk)
			}
		})
	}
}
