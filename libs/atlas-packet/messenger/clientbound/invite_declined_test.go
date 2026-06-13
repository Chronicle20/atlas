package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=messenger/clientbound/MessengerInviteDeclined version=gms_v83 ida=0x8511fc
// packet-audit:verify packet=messenger/clientbound/MessengerInviteDeclined version=gms_v87 ida=0x8b978f
// packet-audit:verify packet=messenger/clientbound/MessengerInviteDeclined version=gms_v95 ida=0x7f51a0
// packet-audit:verify packet=messenger/clientbound/MessengerInviteDeclined version=jms_v185 ida=0x8e4601
// packet-audit:verify packet=messenger/clientbound/MessengerInviteDeclined version=gms_v84 ida=0x87cbd8
func TestMessengerInviteDeclinedRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewMessengerInviteDeclined(7, "TestPlayer", 3)
			output := InviteDeclined{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.Message() != input.Message() {
				t.Errorf("message: got %v, want %v", output.Message(), input.Message())
			}
			if output.DeclineMode() != input.DeclineMode() {
				t.Errorf("declineMode: got %v, want %v", output.DeclineMode(), input.DeclineMode())
			}
		})
	}
}
