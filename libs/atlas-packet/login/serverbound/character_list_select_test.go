package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestCharacterSelectRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := CharacterSelect{
				characterId: 12345,
				mac:         "AA:BB:CC",
				hwid:        "HWID",
			}
			output := CharacterSelect{}
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
