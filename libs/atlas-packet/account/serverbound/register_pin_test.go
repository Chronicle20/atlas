package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=account/serverbound/RegisterPin version=gms_v83 ida=0x5fc89d
// packet-audit:verify packet=account/serverbound/RegisterPin version=gms_v87 ida=0x6342b0
// packet-audit:verify packet=account/serverbound/RegisterPin version=gms_v95 ida=0x5db000
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
