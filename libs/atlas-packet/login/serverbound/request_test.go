package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestRequestRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := Request{
				name:           "testuser",
				password:       "testpass",
				hwid:           make([]byte, 16),
				gameRoomClient: 42,
				gameStartMode:  1,
				unknown1:       2,
				unknown2:       3,
			}
			output := Request{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Name() != input.Name() {
				t.Errorf("name: got %v, want %v", output.Name(), input.Name())
			}
			if output.Password() != input.Password() {
				t.Errorf("password: got %v, want %v", output.Password(), input.Password())
			}
			if output.GameRoomClient() != input.GameRoomClient() {
				t.Errorf("gameRoomClient: got %v, want %v", output.GameRoomClient(), input.GameRoomClient())
			}
			if output.GameStartMode() != input.GameStartMode() {
				t.Errorf("gameStartMode: got %v, want %v", output.GameStartMode(), input.GameStartMode())
			}
		})
	}
}
