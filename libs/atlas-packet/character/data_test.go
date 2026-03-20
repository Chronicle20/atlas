package character

import (
	"testing"
	"time"

	"github.com/Chronicle20/atlas-packet/model"
	pt "github.com/Chronicle20/atlas-packet/test"
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
