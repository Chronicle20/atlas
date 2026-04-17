package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestCreateCharacterRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := CreateCharacter{
				name:             "TestChar",
				jobIndex:         1,
				subJobIndex:      0,
				face:             20000,
				hair:             30000,
				hairColor:        0,
				skinColor:        0,
				topTemplateId:    1040002,
				bottomTemplateId: 1060002,
				shoesTemplateId:  1072001,
				weaponTemplateId: 1302000,
				gender:           0,
				strength:         13,
				dexterity:        4,
				intelligence:     4,
				luck:             4,
			}
			output := CreateCharacter{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Name() != input.Name() {
				t.Errorf("name: got %v, want %v", output.Name(), input.Name())
			}
			if output.Face() != input.Face() {
				t.Errorf("face: got %v, want %v", output.Face(), input.Face())
			}
			if output.Hair() != input.Hair() {
				t.Errorf("hair: got %v, want %v", output.Hair(), input.Hair())
			}
			if output.TopTemplateId() != input.TopTemplateId() {
				t.Errorf("topTemplateId: got %v, want %v", output.TopTemplateId(), input.TopTemplateId())
			}
			if output.BottomTemplateId() != input.BottomTemplateId() {
				t.Errorf("bottomTemplateId: got %v, want %v", output.BottomTemplateId(), input.BottomTemplateId())
			}
			if output.ShoesTemplateId() != input.ShoesTemplateId() {
				t.Errorf("shoesTemplateId: got %v, want %v", output.ShoesTemplateId(), input.ShoesTemplateId())
			}
			if output.WeaponTemplateId() != input.WeaponTemplateId() {
				t.Errorf("weaponTemplateId: got %v, want %v", output.WeaponTemplateId(), input.WeaponTemplateId())
			}
		})
	}
}
