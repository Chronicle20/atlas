package model

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestCharacterStatisticsRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewCharacterStatistics(
				12345, "TestChar", 1, 3, 20001, 30001,
				[3]uint64{100, 200, 300},
				50, 111,
				40, 30, 20, 10,
				5000, 5000, 3000, 3000,
				5, false, 3,
				123456, 100, 5000,
				100000, 2,
			)
			output := CharacterStatistics{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)

			if output.Id() != input.Id() {
				t.Errorf("id: got %v, want %v", output.Id(), input.Id())
			}
			if output.Name() != input.Name() {
				t.Errorf("name: got %v, want %v", output.Name(), input.Name())
			}
			if output.Gender() != input.Gender() {
				t.Errorf("gender: got %v, want %v", output.Gender(), input.Gender())
			}
			if output.SkinColor() != input.SkinColor() {
				t.Errorf("skinColor: got %v, want %v", output.SkinColor(), input.SkinColor())
			}
			if output.Face() != input.Face() {
				t.Errorf("face: got %v, want %v", output.Face(), input.Face())
			}
			if output.Hair() != input.Hair() {
				t.Errorf("hair: got %v, want %v", output.Hair(), input.Hair())
			}
			if output.Level() != input.Level() {
				t.Errorf("level: got %v, want %v", output.Level(), input.Level())
			}
			if output.JobId() != input.JobId() {
				t.Errorf("jobId: got %v, want %v", output.JobId(), input.JobId())
			}
			if output.Strength() != input.Strength() {
				t.Errorf("strength: got %v, want %v", output.Strength(), input.Strength())
			}
			if output.Hp() != input.Hp() {
				t.Errorf("hp: got %v, want %v", output.Hp(), input.Hp())
			}
			if output.Sp() != input.Sp() {
				t.Errorf("sp: got %v, want %v", output.Sp(), input.Sp())
			}
			if output.Experience() != input.Experience() {
				t.Errorf("experience: got %v, want %v", output.Experience(), input.Experience())
			}
			if output.Fame() != input.Fame() {
				t.Errorf("fame: got %v, want %v", output.Fame(), input.Fame())
			}
			if output.MapId() != input.MapId() {
				t.Errorf("mapId: got %v, want %v", output.MapId(), input.MapId())
			}
			if output.SpawnPoint() != input.SpawnPoint() {
				t.Errorf("spawnPoint: got %v, want %v", output.SpawnPoint(), input.SpawnPoint())
			}

			// Pet IDs: only check all 3 for versions that write 3
			if output.PetIds()[0] != input.PetIds()[0] {
				t.Errorf("petIds[0]: got %v, want %v", output.PetIds()[0], input.PetIds()[0])
			}
			if v.MajorVersion > 28 || v.Region == "JMS" {
				if output.PetIds()[1] != input.PetIds()[1] {
					t.Errorf("petIds[1]: got %v, want %v", output.PetIds()[1], input.PetIds()[1])
				}
				if output.PetIds()[2] != input.PetIds()[2] {
					t.Errorf("petIds[2]: got %v, want %v", output.PetIds()[2], input.PetIds()[2])
				}
				if output.GachaponExperience() != input.GachaponExperience() {
					t.Errorf("gachaponExperience: got %v, want %v", output.GachaponExperience(), input.GachaponExperience())
				}
			}
		})
	}
}
