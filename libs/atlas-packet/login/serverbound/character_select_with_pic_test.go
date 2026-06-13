package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=login/serverbound/CharacterSelectWithPic version=gms_v83 ida=0x5f726d
// packet-audit:verify packet=login/serverbound/CharacterSelectWithPic version=gms_v87 ida=0x62e9f6
// packet-audit:verify packet=login/serverbound/CharacterSelectWithPic version=gms_v95 ida=0x5da2a0
// packet-audit:verify packet=login/serverbound/CharacterSelectWithPic version=gms_v84 ida=0x60c1e3
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
