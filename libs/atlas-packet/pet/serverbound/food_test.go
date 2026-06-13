package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=pet/serverbound/PetFood version=gms_v83 ida=0xa09905
// packet-audit:verify packet=pet/serverbound/PetFood version=gms_v87 ida=0xa9f1b1
// packet-audit:verify packet=pet/serverbound/PetFood version=gms_v95 ida=0x9d9f20
// packet-audit:verify packet=pet/serverbound/PetFood version=jms_v185 ida=0xaee58f
func TestFoodRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := Food{updateTime: 100, source: 3, itemId: 2000000}
			output := Food{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
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
