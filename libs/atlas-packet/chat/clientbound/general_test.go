package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestGeneralChatRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := GeneralChat{characterId: 12345, gm: true, message: "hello world", show: false}
			output := GeneralChat{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.CharacterId() != input.CharacterId() {
				t.Errorf("characterId: got %v, want %v", output.CharacterId(), input.CharacterId())
			}
			if output.Gm() != input.Gm() {
				t.Errorf("gm: got %v, want %v", output.Gm(), input.Gm())
			}
			if output.Message() != input.Message() {
				t.Errorf("message: got %v, want %v", output.Message(), input.Message())
			}
			if output.Show() != input.Show() {
				t.Errorf("show: got %v, want %v", output.Show(), input.Show())
			}
		})
	}
}
