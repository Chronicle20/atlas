package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/serverbound/DeleteCharacter version=gms_v83 ida=0x5f7c4a
// packet-audit:verify packet=character/serverbound/DeleteCharacter version=gms_v87 ida=0x62f3d3
// packet-audit:verify packet=character/serverbound/DeleteCharacter version=gms_v95 ida=0x5d53a0
func TestDeleteCharacterRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := DeleteCharacter{
				pic:         "123456",
				dob:         19900101,
				characterId: 12345,
			}
			output := DeleteCharacter{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.CharacterId() != input.CharacterId() {
				t.Errorf("characterId: got %v, want %v", output.CharacterId(), input.CharacterId())
			}
			if v.Region == "GMS" && v.MajorVersion > 82 {
				if output.Pic() != input.Pic() {
					t.Errorf("pic: got %v, want %v", output.Pic(), input.Pic())
				}
			} else if v.Region == "GMS" {
				if output.Dob() != input.Dob() {
					t.Errorf("dob: got %v, want %v", output.Dob(), input.Dob())
				}
			}
		})
	}
}
