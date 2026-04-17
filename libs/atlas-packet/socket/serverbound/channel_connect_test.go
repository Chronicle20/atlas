package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestChannelConnectRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ChannelConnect{
				characterId: 12345,
				machineId:   make([]byte, 16),
				gm:          true,
				unknown1:    false,
				unknown2:    99999,
			}
			output := ChannelConnect{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.CharacterId() != input.CharacterId() {
				t.Errorf("characterId: got %v, want %v", output.CharacterId(), input.CharacterId())
			}
			if output.Gm() != input.Gm() {
				t.Errorf("gm: got %v, want %v", output.Gm(), input.Gm())
			}
			if output.Unknown1() != input.Unknown1() {
				t.Errorf("unknown1: got %v, want %v", output.Unknown1(), input.Unknown1())
			}
			if output.Unknown2() != input.Unknown2() {
				t.Errorf("unknown2: got %v, want %v", output.Unknown2(), input.Unknown2())
			}
		})
	}
}
