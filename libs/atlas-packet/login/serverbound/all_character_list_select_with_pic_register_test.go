package serverbound

import (
	"testing"

	"github.com/Chronicle20/atlas-constants/world"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestAllCharacterListSelectWithPicRegisterRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := AllCharacterListSelectWithPicRegister{
				opt:         1,
				characterId: 12345,
				worldId:     world.Id(2),
				mac:         "AA:BB:CC",
				hwid:        "HWID",
				pic:         "123456",
			}
			output := AllCharacterListSelectWithPicRegister{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Opt() != input.Opt() {
				t.Errorf("opt: got %v, want %v", output.Opt(), input.Opt())
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
			if output.Pic() != input.Pic() {
				t.Errorf("pic: got %v, want %v", output.Pic(), input.Pic())
			}
		})
	}
}
