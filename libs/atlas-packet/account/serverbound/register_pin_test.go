package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestRegisterPinRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name+"/with_pin", func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := RegisterPin{pinInput: true, pin: "1234"}
			output := RegisterPin{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.PinInput() != input.PinInput() {
				t.Errorf("pinInput: got %v, want %v", output.PinInput(), input.PinInput())
			}
			if output.Pin() != input.Pin() {
				t.Errorf("pin: got %v, want %v", output.Pin(), input.Pin())
			}
		})
		t.Run(v.Name+"/no_pin", func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := RegisterPin{pinInput: false}
			output := RegisterPin{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.PinInput() != input.PinInput() {
				t.Errorf("pinInput: got %v, want %v", output.PinInput(), input.PinInput())
			}
		})
	}
}
