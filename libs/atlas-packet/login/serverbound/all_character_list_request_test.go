package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

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
			// Only encoded for GMS > 83
			if v.Region == "GMS" && v.MajorVersion > 83 {
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
