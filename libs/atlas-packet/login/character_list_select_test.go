package login

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestCharacterSelectedRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := CharacterSelected{
				characterId: 12345,
				mac:         "AA:BB:CC",
				hwid:        "HWID",
			}
			output := CharacterSelected{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.CharacterId() != input.CharacterId() {
				t.Errorf("characterId: got %v, want %v", output.CharacterId(), input.CharacterId())
			}
			if v.Region == "GMS" && v.MajorVersion > 12 {
				if output.Mac() != input.Mac() {
					t.Errorf("mac: got %v, want %v", output.Mac(), input.Mac())
				}
				if output.Hwid() != input.Hwid() {
					t.Errorf("hwid: got %v, want %v", output.Hwid(), input.Hwid())
				}
			}
		})
	}
}
