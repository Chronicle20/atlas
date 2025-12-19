package monster

import (
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"testing"
)

func TestSunnyDay(t *testing.T) {
	r := GetMonsterRegistry()
	r.Clear()
	tenant, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	worldId := byte(0)
	channelId := byte(0)
	mapId := uint32(40000)
	monsterId := uint32(9300018)
	x := int16(0)
	y := int16(0)
	fh := int16(0)
	stance := byte(0)
	team := int8(0)
	hp := uint32(50)
	mp := uint32(50)

	m := r.CreateMonster(tenant, worldId, channelId, mapId, monsterId, x, y, fh, stance, team, hp, mp)
	if !valid(worldId, channelId, mapId, monsterId, x, y, fh, stance, team, hp, mp)(m) {
		t.Fatal("Monster created with incorrect properties.")
	}
	if m.UniqueId() != 1000000000 {
		t.Fatal("Unexpected Unique Id.")
	}
	if m.ControlCharacterId() != 0 {
		t.Fatal("Unexpected Control CharacterId.")
	}

	controlId := uint32(100)
	var err error
	m, err = r.ControlMonster(tenant, m.UniqueId(), controlId)
	if err != nil {
		t.Fatalf("Unable to control monster. err %s", err.Error())
	}
	if m.ControlCharacterId() != controlId {
		t.Fatal("Unexpected Control CharacterId.")
	}

	m, err = r.ClearControl(tenant, m.UniqueId())
	if err != nil {
		t.Fatalf("Unable to clear monster control. err %s", err.Error())
	}
	if m.ControlCharacterId() != 0 {
		t.Fatal("Unexpected Control CharacterId.")
	}

	m2 := r.CreateMonster(tenant, worldId, channelId, mapId, monsterId, 50, y, fh, stance, team, hp, mp)
	if !valid(worldId, channelId, mapId, monsterId, 50, y, fh, stance, team, hp, mp)(m2) {
		t.Fatal("Monster created with incorrect properties.")
	}
	m3 := r.CreateMonster(tenant, worldId, channelId, mapId, monsterId, 100, y, fh, stance, team, hp, mp)
	if !valid(worldId, channelId, mapId, monsterId, 100, y, fh, stance, team, hp, mp)(m3) {
		t.Fatal("Monster created with incorrect properties.")
	}

	irm, err := r.GetMonster(tenant, m.UniqueId())
	if err != nil {
		t.Fatalf("Unable to get monster. err %s", err.Error())
	}
	if !compare(irm)(m) {
		t.Fatal("Monster retrieved with incorrect properties.")
	}

	imms := r.GetMonstersInMap(tenant, worldId, channelId, mapId)
	if len(imms) != 3 {
		t.Fatal("Monsters in map not correct.")
	}
	for _, imm := range imms {
		if compare(imm)(m) {
			continue
		}
		if compare(imm)(m2) {
			continue
		}
		if compare(imm)(m3) {
			continue
		}
		t.Fatalf("Monster retrieved with incorrect properties.")
	}

	_, err = r.RemoveMonster(tenant, m.UniqueId())
	if err != nil {
		t.Fatalf("Unable to remove monster. err %s", err.Error())
	}
	imms = r.GetMonstersInMap(tenant, worldId, channelId, mapId)
	if len(imms) != 2 {
		t.Fatal("Monsters in map not correct.")
	}
	for _, imm := range imms {
		if compare(imm)(m2) {
			continue
		}
		if compare(imm)(m3) {
			continue
		}
		t.Fatalf("Monster retrieved with incorrect properties.")
	}
}

func TestIdReuse(t *testing.T) {
	r := GetMonsterRegistry()
	r.Clear()
	tenant1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tenant2, _ := tenant.Create(uuid.New(), "GMS", 87, 1)
	worldId := byte(0)
	channelId := byte(0)
	mapId := uint32(40000)
	monsterId := uint32(9300018)
	x := int16(0)
	y := int16(0)
	fh := int16(0)
	stance := byte(0)
	team := int8(0)
	hp := uint32(50)
	mp := uint32(50)

	m := r.CreateMonster(tenant1, worldId, channelId, mapId, monsterId, x, y, fh, stance, team, hp, mp)
	if !valid(worldId, channelId, mapId, monsterId, x, y, fh, stance, team, hp, mp)(m) {
		t.Fatal("Monster created with incorrect properties.")
	}
	if m.UniqueId() != 1000000000 {
		t.Fatal("Unexpected Unique Id.")
	}

	m2 := r.CreateMonster(tenant2, worldId, channelId, mapId, monsterId, x, y, fh, stance, team, hp, mp)
	if !valid(worldId, channelId, mapId, monsterId, x, y, fh, stance, team, hp, mp)(m) {
		t.Fatal("Monster created with incorrect properties.")
	}
	if m2.UniqueId() != 1000000000 {
		t.Fatal("Unexpected Unique Id.")
	}

	m3 := r.CreateMonster(tenant1, worldId, channelId, mapId, monsterId, x, y, fh, stance, team, hp, mp)
	if !valid(worldId, channelId, mapId, monsterId, x, y, fh, stance, team, hp, mp)(m) {
		t.Fatal("Monster created with incorrect properties.")
	}
	if m3.UniqueId() != 1000000001 {
		t.Fatal("Unexpected Unique Id.")
	}
}

func valid(worldId byte, channelId byte, mapId uint32, monsterId uint32, x int16, y int16, fh int16, stance byte, team int8, hp uint32, mp uint32) func(m Model) bool {
	return func(m Model) bool {
		if m.WorldId() != worldId {
			return false
		}
		if m.ChannelId() != channelId {
			return false
		}
		if m.MapId() != mapId {
			return false
		}
		if m.MonsterId() != monsterId {
			return false
		}
		if m.X() != x {
			return false
		}
		if m.Y() != y {
			return false
		}
		if m.Fh() != fh {
			return false
		}
		if m.Stance() != stance {
			return false
		}
		if m.Team() != team {
			return false
		}
		if m.Hp() != hp {
			return false
		}
		if m.Mp() != mp {
			return false
		}
		return true
	}
}

func compare(m Model) func(o Model) bool {
	return func(o Model) bool {
		if m.UniqueId() != o.UniqueId() {
			return false
		}
		if m.WorldId() != o.WorldId() {
			return false
		}
		if m.ChannelId() != o.ChannelId() {
			return false
		}
		if m.MapId() != o.MapId() {
			return false
		}
		if m.Hp() != o.Hp() {
			return false
		}
		if m.Mp() != o.Mp() {
			return false
		}
		if m.X() != o.X() {
			return false
		}
		if m.Y() != o.Y() {
			return false
		}
		if m.MonsterId() != o.MonsterId() {
			return false
		}
		if m.ControlCharacterId() != o.ControlCharacterId() {
			return false
		}
		return true
	}
}

func TestDestroyAll(t *testing.T) {
	r := GetMonsterRegistry()
	r.Clear()
	tenant1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tenant2, _ := tenant.Create(uuid.New(), "GMS", 87, 1)
	worldId := byte(0)
	channelId := byte(0)
	mapId := uint32(40000)
	monsterId := uint32(9300018)
	x := int16(0)
	y := int16(0)
	fh := int16(0)
	stance := byte(0)
	team := int8(0)
	hp := uint32(50)
	mp := uint32(50)

	_ = r.CreateMonster(tenant1, worldId, channelId, mapId, monsterId, x, y, fh, stance, team, hp, mp)
	_ = r.CreateMonster(tenant2, worldId, channelId, mapId, monsterId, x, y, fh, stance, team, hp, mp)
	_ = r.CreateMonster(tenant1, worldId, channelId, mapId, monsterId, x, y, fh, stance, team, hp, mp)

	ms := r.GetMonsters()
	count := 0
	for _, v := range ms {
		count += len(v)
	}
	if count != 3 {
		t.Fatal("Expected 3 Monsters, got ", count)
	}
}
