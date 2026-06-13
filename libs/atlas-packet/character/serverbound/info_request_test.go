package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/serverbound/InfoRequest version=gms_v83 ida=0xa23fba
// packet-audit:verify packet=character/serverbound/InfoRequest version=gms_v87 ida=0xabba88
// packet-audit:verify packet=character/serverbound/InfoRequest version=gms_v95 ida=0x9f2f70
// packet-audit:verify packet=character/serverbound/InfoRequest version=jms_v185 ida=0xb0b323
// packet-audit:verify packet=character/serverbound/InfoRequest version=gms_v84 ida=0xa6f657
func TestInfoRequestRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := InfoRequest{updateTime: 100, characterId: 12345, petInfo: true}
			output := InfoRequest{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.UpdateTime() != input.UpdateTime() {
				t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
			}
			if output.CharacterId() != input.CharacterId() {
				t.Errorf("characterId: got %v, want %v", output.CharacterId(), input.CharacterId())
			}
			if output.PetInfo() != input.PetInfo() {
				t.Errorf("petInfo: got %v, want %v", output.PetInfo(), input.PetInfo())
			}
		})
	}
}
