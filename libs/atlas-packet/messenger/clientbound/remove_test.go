package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=messenger/clientbound/MessengerRemove version=gms_v83 ida=0x8511fc
// packet-audit:verify packet=messenger/clientbound/MessengerRemove version=gms_v87 ida=0x8b978f
// packet-audit:verify packet=messenger/clientbound/MessengerRemove version=gms_v95 ida=0x7f5e20
// packet-audit:verify packet=messenger/clientbound/MessengerRemove version=jms_v185 ida=0x8e447e
func TestMessengerRemoveRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewMessengerRemove(2, 1)
			output := Remove{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.Position() != input.Position() {
				t.Errorf("position: got %v, want %v", output.Position(), input.Position())
			}
		})
	}
}
