package serverbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=login/serverbound/AllCharacterListSelectWithPic version=gms_v83 ida=0x5f76ae
// packet-audit:verify packet=login/serverbound/AllCharacterListSelectWithPic version=gms_v84 ida=0x60c624
// packet-audit:verify packet=login/serverbound/AllCharacterListSelectWithPic version=gms_v87 ida=0x62ee37
// packet-audit:verify packet=login/serverbound/AllCharacterListSelectWithPic version=gms_v95 ida=0x5d7550
func TestAllCharacterListSelectWithPicRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := AllCharacterListSelectWithPic{
				pic:         "123456",
				characterId: 12345,
				worldId:     world.Id(2),
				mac:         "AA:BB:CC",
				hwid:        "HWID",
			}
			output := AllCharacterListSelectWithPic{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Pic() != input.Pic() {
				t.Errorf("pic: got %v, want %v", output.Pic(), input.Pic())
			}
			if output.CharacterId() != input.CharacterId() {
				t.Errorf("characterId: got %v, want %v", output.CharacterId(), input.CharacterId())
			}
			if output.WorldId() != input.WorldId() {
				t.Errorf("worldId: got %v, want %v", output.WorldId(), input.WorldId())
			}
			if output.Mac() != input.Mac() {
				t.Errorf("mac: got %v, want %v", output.Mac(), input.Mac())
			}
			if output.Hwid() != input.Hwid() {
				t.Errorf("hwid: got %v, want %v", output.Hwid(), input.Hwid())
			}
		})
	}
}
