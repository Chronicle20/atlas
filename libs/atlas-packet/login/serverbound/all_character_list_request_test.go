package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=login/serverbound/AllCharacterListRequest version=gms_v83 ida=0x5fac34
// packet-audit:verify packet=login/serverbound/AllCharacterListRequest version=gms_v87 ida=0x6324e3
// packet-audit:verify packet=login/serverbound/AllCharacterListRequest version=gms_v95 ida=0x5dfb40
// packet-audit:verify packet=login/serverbound/AllCharacterListRequest version=gms_v84 ida=0x60fc30
// packet-audit:verify packet=login/serverbound/AllCharacterListRequest version=jms_v185 ida=0x67094e
func TestAllCharacterListRequestRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := AllCharacterListRequest{
				gameStartMode:  1,
				nexonPassport:  "passport",
				machineId:      make([]byte, 16),
				gameRoomClient: 42,
				gameStartMode2: 2,
			}
			output := AllCharacterListRequest{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			// Only encoded for GMS >= 87 (v84..86 == v83, off-by-one fix, delta §3.2)
			if v.Region == "GMS" && v.MajorVersion >= 87 {
				if output.GameStartMode() != input.GameStartMode() {
					t.Errorf("gameStartMode: got %v, want %v", output.GameStartMode(), input.GameStartMode())
				}
				if output.NexonPassport() != input.NexonPassport() {
					t.Errorf("nexonPassport: got %v, want %v", output.NexonPassport(), input.NexonPassport())
				}
				if output.GameRoomClient() != input.GameRoomClient() {
					t.Errorf("gameRoomClient: got %v, want %v", output.GameRoomClient(), input.GameRoomClient())
				}
				if output.GameStartMode2() != input.GameStartMode2() {
					t.Errorf("gameStartMode2: got %v, want %v", output.GameStartMode2(), input.GameStartMode2())
				}
			}
		})
	}
}
