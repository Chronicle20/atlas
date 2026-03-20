package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestGeneralRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := General{updateTime: 100, msg: "hello world", bOnlyBalloon: true}
			output := General{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Msg() != input.Msg() {
				t.Errorf("msg: got %v, want %v", output.Msg(), input.Msg())
			}
			if output.BOnlyBalloon() != input.BOnlyBalloon() {
				t.Errorf("bOnlyBalloon: got %v, want %v", output.BOnlyBalloon(), input.BOnlyBalloon())
			}
		})
	}
}
