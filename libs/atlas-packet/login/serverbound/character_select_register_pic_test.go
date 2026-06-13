package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=login/serverbound/CharacterSelectRegisterPic version=gms_v83 ida=0x5f726d
// packet-audit:verify packet=login/serverbound/CharacterSelectRegisterPic version=gms_v87 ida=0x62e9f6
// packet-audit:verify packet=login/serverbound/CharacterSelectRegisterPic version=gms_v95 ida=0x5da2a0
func TestCharacterSelectRegisterPicRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := CharacterSelectRegisterPic{
				mode:        1,
				characterId: 12345,
				mac:         "AA:BB:CC:DD:EE:FF",
				hwid:        "HWID123",
				pic:         "123456",
			}
			output := CharacterSelectRegisterPic{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
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
			if output.Pic() != input.Pic() {
				t.Errorf("pic: got %v, want %v", output.Pic(), input.Pic())
			}
		})
	}
}
