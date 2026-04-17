package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func TestCharacterInfoEncode(t *testing.T) {
	pets := []InfoPet{
		{Slot: 0, TemplateId: 5000001, Name: "Kitty", Level: 10, Closeness: 100, Fullness: 50},
	}
	input := NewCharacterInfo(12345, 50, 100, 10, "TestGuild", pets, []uint32{50200004}, 1142007)
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

func TestCharacterInfoRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pets := []InfoPet{
				{Slot: 0, TemplateId: 5000000, Name: "MiniDog", Level: 15, Closeness: 200, Fullness: 80},
				{Slot: 1, TemplateId: 5000001, Name: "MiniCat", Level: 10, Closeness: 100, Fullness: 50},
			}
			wishList := []uint32{1002000, 1002001, 1002002}
			input := NewCharacterInfo(100, 70, 312, 50, "TestGuild", pets, wishList, 1142000)
			output := CharacterInfo{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.CharacterId() != input.CharacterId() {
				t.Errorf("characterId: got %v, want %v", output.CharacterId(), input.CharacterId())
			}
			if output.Level() != input.Level() {
				t.Errorf("level: got %v, want %v", output.Level(), input.Level())
			}
			if output.JobId() != input.JobId() {
				t.Errorf("jobId: got %v, want %v", output.JobId(), input.JobId())
			}
			if output.Fame() != input.Fame() {
				t.Errorf("fame: got %v, want %v", output.Fame(), input.Fame())
			}
			if output.GuildName() != input.GuildName() {
				t.Errorf("guildName: got %v, want %v", output.GuildName(), input.GuildName())
			}
			if len(output.Pets()) != len(input.Pets()) {
				t.Errorf("pets count: got %v, want %v", len(output.Pets()), len(input.Pets()))
			} else {
				for i, p := range output.Pets() {
					if p.TemplateId != pets[i].TemplateId {
						t.Errorf("pet[%d] templateId: got %v, want %v", i, p.TemplateId, pets[i].TemplateId)
					}
					if p.Name != pets[i].Name {
						t.Errorf("pet[%d] name: got %v, want %v", i, p.Name, pets[i].Name)
					}
				}
			}
			if len(output.WishList()) != len(input.WishList()) {
				t.Errorf("wishList count: got %v, want %v", len(output.WishList()), len(input.WishList()))
			}
			if output.MedalId() != input.MedalId() {
				t.Errorf("medalId: got %v, want %v", output.MedalId(), input.MedalId())
			}
		})
	}
}

func TestCharacterInfoEmptyRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewCharacterInfo(200, 30, 100, 0, "", nil, nil, 0)
			output := CharacterInfo{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if len(output.Pets()) != 0 {
				t.Errorf("pets count: got %v, want 0", len(output.Pets()))
			}
			if len(output.WishList()) != 0 {
				t.Errorf("wishList count: got %v, want 0", len(output.WishList()))
			}
		})
	}
}
