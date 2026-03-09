package login

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestCharacterSelectWithPicRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := CharacterSelectWithPic{
				pic:         "123456",
				characterId: 12345,
				mac:         "AA:BB:CC",
				hwid:        "HWID",
			}
			output := CharacterSelectWithPic{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Pic() != input.Pic() {
				t.Errorf("pic: got %v, want %v", output.Pic(), input.Pic())
			}
			if output.CharacterId() != input.CharacterId() {
				t.Errorf("characterId: got %v, want %v", output.CharacterId(), input.CharacterId())
			}
			if v.Region == "GMS" {
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
