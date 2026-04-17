package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

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
