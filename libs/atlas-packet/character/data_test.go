package character

import (
	"bytes"
	"strconv"
	"testing"
	"time"

	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestEncodeSkillsExpirationVersionGate guards the v79 field-enter regression:
// the per-skill Int64 expiration is v83+ only. pt.Variants has no version in
// [29,82], so this pins the boundary directly. A legacy (<83) skill entry is
// id(4)+level(4); v83+ adds the 8-byte expiration. Both write the cooldown
// count short (GMS>28).
func TestEncodeSkillsExpirationVersionGate(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	cd := CharacterData{Skills: []SkillEntry{{Id: 2101001, Level: 20, Expiration: -1}}}

	// v79 (legacy): count(2) + id(4) + level(4) + cooldownCount(2) = 12, no expiration.
	w79 := response.NewWriter(l)
	cd.encodeSkills(w79, tenant.MustFromContext(pt.CreateContext("GMS", 79, 1)))
	if got := len(w79.Bytes()); got != 12 {
		t.Errorf("v79 encodeSkills = %d bytes, want 12 (id+level, no Int64 expiration)", got)
	}

	// v83+: adds the 8-byte Int64 expiration = 20.
	w83 := response.NewWriter(l)
	cd.encodeSkills(w83, tenant.MustFromContext(pt.CreateContext("GMS", 83, 1)))
	if got := len(w83.Bytes()); got != 20 {
		t.Errorf("v83 encodeSkills = %d bytes, want 20 (id+level+expiration)", got)
	}
}

// TestCharacterDataLegacyFieldGate_V72 guards the v72 field-enter crash: three
// fields were added to CharacterData in the v79 protocol revision and are ABSENT
// in v48/v61/v72 — the SN-list-size byte (after the 8-byte flag), the linked-name
// byte (between buddyCap and meso), and the 8-byte inventory-update FILETIME
// (before the equip section). IDA-verified against CharacterData::Decode v72
// @0x4d1c60 vs v79 @0x4d9b85 / v83 @0x4e592d. With no inventory items the two
// encodes differ by EXACTLY those 10 bytes; encoding them for v72 shifts the whole
// stream (the SN byte lands as the first byte of the character id) and crashes the
// client on channel entry.
func TestCharacterDataLegacyFieldGate_V72(t *testing.T) {
	cd := CharacterData{
		Stats: CharacterStats{
			Id: 1000, Name: "TestChar", Gender: 0, SkinColor: 1,
			Face: 20000, Hair: 30000,
			Level: 50, JobId: 312, Str: 100, Dex: 50, Int: 30, Luk: 20,
			Hp: 5000, MaxHp: 5000, Mp: 3000, MaxMp: 3000,
			Ap: 5, Sp: 3, Exp: 50000, Fame: 10,
			MapId: 100000000, SpawnPoint: 0,
		},
		BuddyCapacity: 20,
		Meso:          100000,
		Inventory: InventoryData{
			EquipCapacity: 24, UseCapacity: 24, SetupCapacity: 24,
			EtcCapacity: 24, CashCapacity: 24,
			Timestamp: 94354848000000000,
		},
	}

	v72 := pt.Encode(t, pt.CreateContext("GMS", 72, 1), cd.Encode, nil)
	v79 := pt.Encode(t, pt.CreateContext("GMS", 79, 1), cd.Encode, nil)

	// SN byte (1) + linked-name byte (1) + inventory FILETIME (8) = 10.
	if diff := len(v79) - len(v72); diff != 10 {
		t.Errorf("v79 CharacterData should be 10 bytes longer than v72 (SN+linked+timestamp); got diff %d (v72=%d v79=%d)", diff, len(v72), len(v79))
	}

	// The 8-byte dbcharFlag (Int64) must be followed immediately by the stat block,
	// i.e. the low byte of stats.Id (1000 = 0x03E8 -> 0xE8), NOT a stray SN byte (0x00).
	if v72[8] != 0xE8 {
		t.Errorf("v72 CharacterData byte[8] = 0x%02X, want 0xE8 (stats.Id low byte); a 0x00 here means the v79-only SN byte leaked into v72", v72[8])
	}
}

// legacySampleCD builds a CharacterData with stats, capacities, and one equipped
// item so the legacy tests exercise the stat tail, the equip trailer, and the
// monster book together.
func legacySampleCD() CharacterData {
	equip := model.NewAsset(false, -5, 1302000, time.Time{}).
		SetEquipmentStats(10, 11, 12, 13, 100, 50, 80, 70, 30, 25, 15, 20, 10, 5, 3).
		SetEquipmentMeta(7, 1, 2, 500, 3, 0x0001)
	return CharacterData{
		Stats: CharacterStats{
			Id: 1000, Name: "TestChar", Gender: 0, SkinColor: 1,
			Face: 20000, Hair: 30000,
			Level: 50, JobId: 312, Str: 100, Dex: 50, Int: 30, Luk: 20,
			Hp: 5000, MaxHp: 5000, Mp: 3000, MaxMp: 3000,
			Ap: 5, Sp: 3, Exp: 50000, Fame: 10, GachaExp: 777,
			MapId: 100000000, SpawnPoint: 0,
		},
		BuddyCapacity: 20,
		Meso:          100000,
		Inventory: InventoryData{
			EquipCapacity: 24, UseCapacity: 24, SetupCapacity: 24,
			EtcCapacity: 24, CashCapacity: 24,
			Timestamp:    94354848000000000,
			RegularEquip: []model.Asset{equip},
		},
		MonsterBook: MonsterBookData{CoverCardId: 2388000},
	}
}

// TestCharacterDataLegacyRoundTrip exercises encode→decode symmetry for the
// legacy versions (v48/v61/v72), which pt.Variants does not cover. A mismatch
// here means the decode gates drifted from the encode gates for that version.
func TestCharacterDataLegacyRoundTrip(t *testing.T) {
	for _, major := range []uint16{48, 61, 72, 79, 83} {
		major := major
		t.Run("GMS_v"+strconv.Itoa(int(major)), func(t *testing.T) {
			ctx := pt.CreateContext("GMS", major, 1)
			input := legacySampleCD()
			output := CharacterData{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Stats.Id != input.Stats.Id {
				t.Errorf("stats id: got %d want %d", output.Stats.Id, input.Stats.Id)
			}
			if output.Stats.Name != input.Stats.Name {
				t.Errorf("name: got %q want %q", output.Stats.Name, input.Stats.Name)
			}
			if output.Stats.MapId != input.Stats.MapId {
				t.Errorf("mapId: got %d want %d", output.Stats.MapId, input.Stats.MapId)
			}
			if output.Meso != input.Meso {
				t.Errorf("meso: got %d want %d", output.Meso, input.Meso)
			}
			if len(output.Inventory.RegularEquip) != 1 {
				t.Fatalf("equip count: got %d want 1", len(output.Inventory.RegularEquip))
			}
			if output.Inventory.RegularEquip[0].TemplateId() != 1302000 {
				t.Errorf("equip templateId: got %d want 1302000", output.Inventory.RegularEquip[0].TemplateId())
			}
			if output.Inventory.RegularEquip[0].Strength() != 10 {
				t.Errorf("equip str: got %d want 10", output.Inventory.RegularEquip[0].Strength())
			}
			// Tier-gated fields: assert each decodes back only for the versions
			// that carry it, so a decode gate drifting from its encode gate fails
			// with a value mismatch (not just a leftover-byte error).
			// Monster-book cover is v61+.
			wantCover := item.Id(0)
			if major >= 61 {
				wantCover = input.MonsterBook.CoverCardId
			}
			if output.MonsterBook.CoverCardId != wantCover {
				t.Errorf("monster-book cover: got %d want %d (v%d)", output.MonsterBook.CoverCardId, wantCover, major)
			}
			// Equip experience is v72+; hammersApplied is v79+.
			wantExp := uint32(0)
			if major >= 72 {
				wantExp = 500
			}
			if output.Inventory.RegularEquip[0].Experience() != wantExp {
				t.Errorf("equip experience: got %d want %d (v%d)", output.Inventory.RegularEquip[0].Experience(), wantExp, major)
			}
			wantHammers := uint32(0)
			if major >= 79 {
				wantHammers = 3
			}
			if output.Inventory.RegularEquip[0].HammersApplied() != wantHammers {
				t.Errorf("equip hammersApplied: got %d want %d (v%d)", output.Inventory.RegularEquip[0].HammersApplied(), wantHammers, major)
			}
		})
	}
}

// TestCharacterDataLegacyStructure pins the two legacy boundaries that reshape the
// packet framing: the dbcharFlag widened from Int16 to Int64 at v61, and the total
// length grows monotonically v48 < v61 < v72 < v79 as each revision adds sections.
func TestCharacterDataLegacyStructure(t *testing.T) {
	cd := legacySampleCD()
	enc := func(major uint16) []byte { return pt.Encode(t, pt.CreateContext("GMS", major, 1), cd.Encode, nil) }
	v48, v61, v72, v79 := enc(48), enc(61), enc(72), enc(79)

	// dbcharFlag width: v48 is a 2-byte Int16 (stat block, i.e. stats.Id low byte
	// 0xE8, follows immediately); v61+ is an 8-byte Int64.
	if v48[2] != 0xE8 {
		t.Errorf("v48 byte[2] = 0x%02X, want 0xE8 (stats.Id after a 2-byte flag)", v48[2])
	}
	if v61[8] != 0xE8 {
		t.Errorf("v61 byte[8] = 0x%02X, want 0xE8 (stats.Id after an 8-byte flag)", v61[8])
	}

	if len(v48) >= len(v61) || len(v61) >= len(v72) || len(v72) >= len(v79) {
		t.Errorf("expected strictly increasing lengths v48<v61<v72<v79; got %d %d %d %d", len(v48), len(v61), len(v72), len(v79))
	}
}

func TestCharacterDataMinimalRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := CharacterData{
				Stats: CharacterStats{
					Id: 1000, Name: "TestChar", Gender: 0, SkinColor: 1,
					Face: 20000, Hair: 30000,
					Level: 50, JobId: 312, Str: 100, Dex: 50, Int: 30, Luk: 20,
					Hp: 5000, MaxHp: 5000, Mp: 3000, MaxMp: 3000,
					Ap: 5, Sp: 3, Exp: 50000, Fame: 10,
					MapId: 100000000, SpawnPoint: 0,
				},
				BuddyCapacity: 20,
				Meso:          100000,
				Inventory: InventoryData{
					EquipCapacity: 24, UseCapacity: 24, SetupCapacity: 24,
					EtcCapacity: 24, CashCapacity: 24,
					Timestamp: 94354848000000000,
				},
			}
			output := CharacterData{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Stats.Id != input.Stats.Id {
				t.Errorf("stats id: got %v, want %v", output.Stats.Id, input.Stats.Id)
			}
			if output.Stats.Name != input.Stats.Name {
				t.Errorf("stats name: got %q, want %q", output.Stats.Name, input.Stats.Name)
			}
			if output.Stats.Level != input.Stats.Level {
				t.Errorf("stats level: got %v, want %v", output.Stats.Level, input.Stats.Level)
			}
			if output.Meso != input.Meso {
				t.Errorf("meso: got %v, want %v", output.Meso, input.Meso)
			}
			if output.BuddyCapacity != input.BuddyCapacity {
				t.Errorf("buddyCapacity: got %v, want %v", output.BuddyCapacity, input.BuddyCapacity)
			}
		})
	}
}

func TestCharacterDataWithSkillsRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := CharacterData{
				Stats: CharacterStats{
					Id: 2000, Name: "Mage", Level: 120, JobId: 212,
					MapId: 100000000,
				},
				BuddyCapacity: 50,
				Inventory: InventoryData{
					EquipCapacity: 24, UseCapacity: 24, SetupCapacity: 24,
					EtcCapacity: 24, CashCapacity: 24,
					Timestamp: 94354848000000000,
				},
				Skills: []SkillEntry{
					{Id: 2101001, Level: 20, Expiration: -1, FourthJob: false},
					{Id: 2121006, Level: 30, Expiration: -1, FourthJob: true, MasterLevel: 30},
				},
			}
			output := CharacterData{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if len(output.Skills) != len(input.Skills) {
				t.Fatalf("skills count: got %v, want %v", len(output.Skills), len(input.Skills))
			}
			for i, s := range output.Skills {
				if s.Id != input.Skills[i].Id {
					t.Errorf("skill[%d] id: got %v, want %v", i, s.Id, input.Skills[i].Id)
				}
				if s.FourthJob != input.Skills[i].FourthJob {
					t.Errorf("skill[%d] fourthJob: got %v, want %v", i, s.FourthJob, input.Skills[i].FourthJob)
				}
				if s.FourthJob && s.MasterLevel != input.Skills[i].MasterLevel {
					t.Errorf("skill[%d] masterLevel: got %v, want %v", i, s.MasterLevel, input.Skills[i].MasterLevel)
				}
			}
		})
	}
}

func TestCharacterDataWithQuestsRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := CharacterData{
				Stats: CharacterStats{Id: 3000, Name: "QuestChar", MapId: 100000000},
				Inventory: InventoryData{
					EquipCapacity: 24, UseCapacity: 24, SetupCapacity: 24,
					EtcCapacity: 24, CashCapacity: 24,
					Timestamp: 94354848000000000,
				},
				StartedQuests: []QuestProgress{
					{QuestId: 1000, Progress: "001"},
					{QuestId: 1001, Progress: ""},
				},
				CompletedQuests: []QuestCompleted{
					{QuestId: 500, CompletedAt: model.MsTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))},
				},
			}
			output := CharacterData{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if len(output.StartedQuests) != len(input.StartedQuests) {
				t.Errorf("started quests: got %v, want %v", len(output.StartedQuests), len(input.StartedQuests))
			}
			// Completed quests only for GMS>12 or JMS
			if v.MajorVersion > 12 || v.Region == "JMS" {
				if len(output.CompletedQuests) != len(input.CompletedQuests) {
					t.Errorf("completed quests: got %v, want %v", len(output.CompletedQuests), len(input.CompletedQuests))
				}
			}
		})
	}
}

func TestEncodeMonsterBook_Empty(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	w := response.NewWriter(l)
	cd := CharacterData{}
	cd.encodeMonsterBookCover(w)
	cd.encodeMonsterBookCards(w)
	got := w.Bytes()
	// cover int(0) | mode byte(0) | count short(0) — byte-identical to the old stub.
	want := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	if !bytes.Equal(got, want) {
		t.Errorf("empty book bytes = % x, want % x", got, want)
	}
}

func TestEncodeMonsterBook_Populated(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	w := response.NewWriter(l)
	cd := CharacterData{
		MonsterBook: MonsterBookData{
			CoverCardId: 2380001,
			Cards: []MonsterBookCard{
				{CardId: 2380005, Level: 2},
				{CardId: 2382000, Level: 5},
			},
		},
	}
	cd.encodeMonsterBookCover(w)
	cd.encodeMonsterBookCards(w)
	got := w.Bytes()
	// cover 2380001 (LE E1 50 24 00) | mode 00 | count 2 (02 00)
	// | card 5 (05 00) lvl 2 (02) | card 2000 (D0 07) lvl 5 (05)
	want := []byte{
		0xE1, 0x50, 0x24, 0x00,
		0x00,
		0x02, 0x00,
		0x05, 0x00, 0x02,
		0xD0, 0x07, 0x05,
	}
	if !bytes.Equal(got, want) {
		t.Errorf("populated book bytes = % x, want % x", got, want)
	}
}

func TestCharacterDataMonsterBookRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := CharacterData{
				Stats: CharacterStats{Id: 3000, Name: "Booker", Level: 30, JobId: 100, MapId: 100000000},
				Inventory: InventoryData{
					EquipCapacity: 24, UseCapacity: 24, SetupCapacity: 24,
					EtcCapacity: 24, CashCapacity: 24, Timestamp: 94354848000000000,
				},
				MonsterBook: MonsterBookData{
					CoverCardId: 2380001,
					Cards: []MonsterBookCard{
						{CardId: 2380005, Level: 2},
						{CardId: 2382000, Level: 5},
					},
				},
			}
			output := CharacterData{}
			// RoundTrip fails if any byte is left unconsumed — the gate-alignment guard.
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)

			bookPresent := (v.Region == "GMS" && v.MajorVersion > 28 && v.MajorVersion <= 87) || v.Region == "JMS"
			if bookPresent {
				if output.MonsterBook.CoverCardId != input.MonsterBook.CoverCardId {
					t.Errorf("cover: got %d, want %d", output.MonsterBook.CoverCardId, input.MonsterBook.CoverCardId)
				}
				if len(output.MonsterBook.Cards) != len(input.MonsterBook.Cards) {
					t.Fatalf("card count: got %d, want %d", len(output.MonsterBook.Cards), len(input.MonsterBook.Cards))
				}
				for i := range output.MonsterBook.Cards {
					if output.MonsterBook.Cards[i] != input.MonsterBook.Cards[i] {
						t.Errorf("card[%d]: got %+v, want %+v", i, output.MonsterBook.Cards[i], input.MonsterBook.Cards[i])
					}
				}
			} else {
				// v95: monster book absent — encoder wrote nothing, decoder read nothing.
				if output.MonsterBook.CoverCardId != 0 || len(output.MonsterBook.Cards) != 0 {
					t.Errorf("expected empty monster book for %s, got cover=%d cards=%d",
						v.Name, output.MonsterBook.CoverCardId, len(output.MonsterBook.Cards))
				}
			}
		})
	}
}

// FR-15: the teleport region carries real lists; empty slots still encode
// EmptyMapId; the VIP block keeps its (GMS>28)||JMS gate. Decode strips
// padding so round-trip is stable on the canonical (unpadded) form.
func TestCharacterDataTeleportRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := CharacterData{
				Stats: CharacterStats{
					Id: 1000, Name: "TestChar", SkinColor: 1,
					Face: 20000, Hair: 30000, Level: 50, JobId: 312,
					MapId: 100000000,
				},
				Inventory: InventoryData{
					EquipCapacity: 24, UseCapacity: 24, SetupCapacity: 24,
					EtcCapacity: 24, CashCapacity: 24,
					Timestamp: 94354848000000000,
				},
				TeleportMaps:    []_map.Id{100000000, 220000000},
				VipTeleportMaps: []_map.Id{104040000},
			}
			output := CharacterData{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if len(output.TeleportMaps) != 2 || output.TeleportMaps[0] != 100000000 || output.TeleportMaps[1] != 220000000 {
				t.Errorf("teleportMaps: got %v", output.TeleportMaps)
			}
			vipExpected := (v.Region == "GMS" && v.MajorVersion > 28) || v.Region == "JMS"
			if vipExpected {
				if len(output.VipTeleportMaps) != 1 || output.VipTeleportMaps[0] != 104040000 {
					t.Errorf("vipTeleportMaps: got %v", output.VipTeleportMaps)
				}
			} else if len(output.VipTeleportMaps) != 0 {
				t.Errorf("vip block must be absent for %s: got %v", v.Name, output.VipTeleportMaps)
			}
		})
	}
}
