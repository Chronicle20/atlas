package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestCharacterSpawnVersionBoundary pins the corrected >83 -> >=87 boundary
// (delta §3.1.7): v84..86 must encode byte-identically to v83 (no
// nCompletedSetItemID int). v87/v95 stay on the v87+ path.
func TestCharacterSpawnVersionBoundary(t *testing.T) {
	avatar := testSpawnAvatar()
	cts := model.NewCharacterTemporaryStat()
	guild := GuildEmblem{Name: "TestGuild", LogoBackground: 1, LogoBackgroundColor: 2, Logo: 3, LogoColor: 4}
	m := NewCharacterSpawn(12345, 50, "TestChar", guild, cts, 312, avatar, nil, false, 100, 200, 3)
	encode := func(major uint16) []byte {
		ctx := pt.CreateContext("GMS", major, 1)
		return pt.Encode(t, ctx, m.Encode, nil)
	}
	v83 := encode(83)
	for _, major := range []uint16{84, 85, 86} {
		if got := encode(major); !bytes.Equal(got, v83) {
			t.Errorf("CharacterSpawn v%d encode differs from v83 (len %d vs %d); v84..86 must match v83", major, len(got), len(v83))
		}
	}
	if v87 := encode(87); bytes.Equal(v87, v83) {
		t.Errorf("CharacterSpawn v87 must stay on the v87+ path, not equal v83")
	}
	if v95 := encode(95); bytes.Equal(v95, v83) {
		t.Errorf("CharacterSpawn v95 must stay on the v87+ path, not equal v83")
	}
}

// TestCharacterInfoVersionBoundary pins the corrected >83 -> >=87 boundary
// (delta §3.1.11): v84..86 must encode byte-identically to v83 (no trailing
// chair int). v87/v95 stay on the v87+ path.
func TestCharacterInfoVersionBoundary(t *testing.T) {
	pets := []InfoPet{
		{Slot: 0, TemplateId: 5000001, Name: "Kitty", Level: 10, Closeness: 100, Fullness: 50},
	}
	m := NewCharacterInfo(12345, 50, 100, 10, "TestGuild", pets, []uint32{50200004}, 1142007, MonsterBookInfo{})
	encode := func(major uint16) []byte {
		ctx := pt.CreateContext("GMS", major, 1)
		return pt.Encode(t, ctx, m.Encode, nil)
	}
	v83 := encode(83)
	for _, major := range []uint16{84, 85, 86} {
		if got := encode(major); !bytes.Equal(got, v83) {
			t.Errorf("CharacterInfo v%d encode differs from v83 (len %d vs %d); v84..86 must match v83", major, len(got), len(v83))
		}
	}
	if v87 := encode(87); bytes.Equal(v87, v83) {
		t.Errorf("CharacterInfo v87 must stay on the v87+ path, not equal v83")
	}
}
