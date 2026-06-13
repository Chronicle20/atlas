package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=messenger/clientbound/MessengerRequestInvite version=gms_v83 ida=0x8511fc
// packet-audit:verify packet=messenger/clientbound/MessengerRequestInvite version=gms_v87 ida=0x8b978f
// packet-audit:verify packet=messenger/clientbound/MessengerRequestInvite version=gms_v95 ida=0x7f2cb0
// packet-audit:verify packet=messenger/clientbound/MessengerRequestInvite version=jms_v185 ida=0x8e46f2
// packet-audit:verify packet=messenger/clientbound/MessengerRequestInvite version=gms_v84 ida=0x87cbd8
func TestMessengerRequestInviteRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewMessengerRequestInvite(5, "TestPlayer", 12345)
			output := RequestInvite{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.FromName() != input.FromName() {
				t.Errorf("fromName: got %v, want %v", output.FromName(), input.FromName())
			}
			if output.MessengerId() != input.MessengerId() {
				t.Errorf("messengerId: got %v, want %v", output.MessengerId(), input.MessengerId())
			}
		})
	}
}
