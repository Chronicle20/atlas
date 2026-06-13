package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=pet/serverbound/PetItemUse version=gms_v83 ida=0xa0955c
// packet-audit:verify packet=pet/serverbound/PetItemUse version=gms_v87 ida=0xa9ee08
// packet-audit:verify packet=pet/serverbound/PetItemUse version=gms_v95 ida=0x9de400
// packet-audit:verify packet=pet/serverbound/PetItemUse version=jms_v185 ida=0xaee1d4
func TestItemUseRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ItemUse{petId: 12345, buffSkill: true, updateTime: 100, source: 5, itemId: 2000001}
			output := ItemUse{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.PetId() != input.PetId() {
				t.Errorf("petId: got %v, want %v", output.PetId(), input.PetId())
			}
			if output.BuffSkill() != input.BuffSkill() {
				t.Errorf("buffSkill: got %v, want %v", output.BuffSkill(), input.BuffSkill())
			}
			if output.UpdateTime() != input.UpdateTime() {
				t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
			}
			if output.Source() != input.Source() {
				t.Errorf("source: got %v, want %v", output.Source(), input.Source())
			}
			if output.ItemId() != input.ItemId() {
				t.Errorf("itemId: got %v, want %v", output.ItemId(), input.ItemId())
			}
		})
	}
}
