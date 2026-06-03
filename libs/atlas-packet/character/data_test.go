package character

import (
	"bytes"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

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
	cd.encodeMonsterBook(w)
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
	cd.encodeMonsterBook(w)
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
