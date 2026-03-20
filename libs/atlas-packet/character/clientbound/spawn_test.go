package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas-packet/model"
	pt "github.com/Chronicle20/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func TestCharacterSpawnEncode(t *testing.T) {
	avatar := model.Avatar{}
	cts := model.NewCharacterTemporaryStat()
	guild := GuildEmblem{Name: "TestGuild"}
	input := NewCharacterSpawn(12345, 50, "TestChar", guild, cts, 100, avatar, nil, true, 100, 200, 6)
	l, _ := testlog.NewNullLogger()
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			encoded := input.Encode(l, ctx)(nil)
			if len(encoded) == 0 {
				t.Error("expected non-empty encoded bytes")
			}
		})
	}
}

func testSpawnAvatar() model.Avatar {
	equip := map[slot.Position]uint32{5: 1040002, 6: 1060002, 7: 1072001}
	masked := map[slot.Position]uint32{}
	pets := map[int8]uint32{}
	return model.NewAvatar(0, 1, 20000, false, 30000, equip, masked, pets)
}

func TestCharacterSpawnRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			avatar := testSpawnAvatar()
			cts := model.NewCharacterTemporaryStat()
			guild := GuildEmblem{Name: "TestGuild", LogoBackground: 1, LogoBackgroundColor: 2, Logo: 3, LogoColor: 4}
			// enteringField=false for exact round-trip
			input := NewCharacterSpawn(12345, 50, "TestChar", guild, cts, 312, avatar, nil, false, 100, 200, 3)
			output := CharacterSpawn{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.CharacterId() != input.CharacterId() {
				t.Errorf("characterId: got %v, want %v", output.CharacterId(), input.CharacterId())
			}
			if output.Level() != input.Level() {
				t.Errorf("level: got %v, want %v", output.Level(), input.Level())
			}
			if output.Name() != input.Name() {
				t.Errorf("name: got %v, want %v", output.Name(), input.Name())
			}
			if output.Guild().Name != input.Guild().Name {
				t.Errorf("guildName: got %v, want %v", output.Guild().Name, input.Guild().Name)
			}
			if output.JobId() != input.JobId() {
				t.Errorf("jobId: got %v, want %v", output.JobId(), input.JobId())
			}
			if output.X() != input.X() {
				t.Errorf("x: got %v, want %v", output.X(), input.X())
			}
			if output.Y() != input.Y() {
				t.Errorf("y: got %v, want %v", output.Y(), input.Y())
			}
			if output.Stance() != input.Stance() {
				t.Errorf("stance: got %v, want %v", output.Stance(), input.Stance())
			}
		})
	}
}

func TestCharacterSpawnWithPetsRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			avatar := testSpawnAvatar()
			cts := model.NewCharacterTemporaryStat()
			guild := GuildEmblem{Name: "Guild"}
			pets := []SpawnPet{
				{Slot: 0, Pet: model.Pet{TemplateId: 5000001, Name: "Dog", Id: 100, X: 10, Y: 20, Stance: 1, Foothold: 5}},
				{Slot: 1, Pet: model.Pet{TemplateId: 5000002, Name: "Cat", Id: 200, X: 30, Y: 40, Stance: 2, Foothold: 6}},
			}
			input := NewCharacterSpawn(999, 80, "PetOwner", guild, cts, 100, avatar, pets, false, 50, 60, 4)
			output := CharacterSpawn{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if len(output.Pets()) != len(input.Pets()) {
				t.Errorf("pets count: got %v, want %v", len(output.Pets()), len(input.Pets()))
			} else {
				for i, p := range output.Pets() {
					if p.Pet.TemplateId != pets[i].Pet.TemplateId {
						t.Errorf("pet[%d] templateId: got %v, want %v", i, p.Pet.TemplateId, pets[i].Pet.TemplateId)
					}
				}
			}
		})
	}
}
