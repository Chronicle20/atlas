package serverbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=login/serverbound/AllCharacterListSelect version=gms_v83 ida=0x5f76ae
// packet-audit:verify packet=login/serverbound/AllCharacterListSelect version=gms_v95 ida=0x5d7550
func TestAllCharacterListSelectRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := AllCharacterListSelect{
				characterId: 12345,
				worldId:     world.Id(2),
				mac:         "AA:BB:CC",
				hwid:        "HWID",
			}
			output := AllCharacterListSelect{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
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
