package clientbound

import (
	"testing"

	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestSetFieldRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			cd := charpkt.CharacterData{
				Stats: charpkt.CharacterStats{
					Id: 1000, Name: "TestChar", Gender: 0, SkinColor: 1,
					Face: 20000, Hair: 30000,
					Level: 50, JobId: 312, Str: 100, Dex: 50, Int: 30, Luk: 20,
					Hp: 5000, MaxHp: 5000, Mp: 3000, MaxMp: 3000,
					Ap: 5, Sp: 3, Exp: 50000, Fame: 10,
					MapId: 100000000, SpawnPoint: 0,
				},
				BuddyCapacity: 20,
				Meso:          100000,
				Inventory: charpkt.InventoryData{
					EquipCapacity: 24, UseCapacity: 24, SetupCapacity: 24,
					EtcCapacity: 24, CashCapacity: 24,
					Timestamp: 94354848000000000,
				},
			}
			input := NewSetField(channel.Id(1), cd)
			output := SetField{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.CharacterData().Stats.Id != cd.Stats.Id {
				t.Errorf("stats id: got %v, want %v", output.CharacterData().Stats.Id, cd.Stats.Id)
			}
			if output.CharacterData().Stats.Name != cd.Stats.Name {
				t.Errorf("stats name: got %q, want %q", output.CharacterData().Stats.Name, cd.Stats.Name)
			}
			if output.CharacterData().Meso != cd.Meso {
				t.Errorf("meso: got %v, want %v", output.CharacterData().Meso, cd.Meso)
			}
		})
	}
}
