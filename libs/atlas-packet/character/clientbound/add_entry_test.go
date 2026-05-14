package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestAddCharacterEntryRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)

			stats := model.NewCharacterStatistics(
				12345, "TestChar", 1, 3, 20001, 30001,
				[3]uint64{100, 200, 300},
				50, 111,
				40, 30, 20, 10,
				5000, 5000, 3000, 3000,
				5, false, 3,
				123456, 100, 5000,
				100000, 2,
			)
			avatar := model.NewAvatar(1, 3, 20001, false, 30001, nil, nil, nil)
			entry := model.NewCharacterListEntry(stats, avatar, false, false, 10, 1, 5, 2)

			input := NewAddCharacterEntry(0, entry)
			output := AddCharacterEntry{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)

			if output.Code() != input.Code() {
				t.Errorf("code: got %v, want %v", output.Code(), input.Code())
			}
			if output.Character().Statistics().Id() != input.Character().Statistics().Id() {
				t.Errorf("characterId: got %v, want %v", output.Character().Statistics().Id(), input.Character().Statistics().Id())
			}
			if output.Character().Statistics().Name() != input.Character().Statistics().Name() {
				t.Errorf("name: got %v, want %v", output.Character().Statistics().Name(), input.Character().Statistics().Name())
			}
			// GMS v28 Avatar.Encode skips gender/skin/face/hair (written only by CharacterStatistics in that version).
			if !(v.Region == "GMS" && v.MajorVersion <= 28) {
				if output.Character().Avatar().Gender() != input.Character().Avatar().Gender() {
					t.Errorf("gender: got %v, want %v", output.Character().Avatar().Gender(), input.Character().Avatar().Gender())
				}
			}
		})
	}
}
