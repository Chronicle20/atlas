package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=pet/serverbound/PetSpawn version=gms_v87 ida=0xabbb70
// packet-audit:verify packet=pet/serverbound/PetSpawn version=gms_v95 ida=0x9f6980
// packet-audit:verify packet=pet/serverbound/PetSpawn version=jms_v185 ida=0xb0b40b
func TestSpawnRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := Spawn{updateTime: 100, slot: -5, lead: true}
			output := Spawn{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.UpdateTime() != input.UpdateTime() {
				t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
			}
			if output.Slot() != input.Slot() {
				t.Errorf("slot: got %v, want %v", output.Slot(), input.Slot())
			}
			if output.Lead() != input.Lead() {
				t.Errorf("lead: got %v, want %v", output.Lead(), input.Lead())
			}
		})
	}
}
