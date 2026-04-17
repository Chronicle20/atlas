package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestAfterLoginRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name+"/with_pin", func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := AfterLogin{pinMode: 1, opt2: 2, pin: "1234"}
			output := AfterLogin{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.PinMode() != input.PinMode() {
				t.Errorf("pinMode: got %v, want %v", output.PinMode(), input.PinMode())
			}
			if output.Opt2() != input.Opt2() {
				t.Errorf("opt2: got %v, want %v", output.Opt2(), input.Opt2())
			}
			if output.Pin() != input.Pin() {
				t.Errorf("pin: got %v, want %v", output.Pin(), input.Pin())
			}
		})
		t.Run(v.Name+"/no_pin", func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := AfterLogin{pinMode: 0}
			output := AfterLogin{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.PinMode() != input.PinMode() {
				t.Errorf("pinMode: got %v, want %v", output.PinMode(), input.PinMode())
			}
		})
	}
}
