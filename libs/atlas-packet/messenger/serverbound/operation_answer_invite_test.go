package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=messenger/serverbound/MessengerOperationAnswerInvite version=gms_v83 ida=0x8511fc
// packet-audit:verify packet=messenger/serverbound/MessengerOperationAnswerInvite version=gms_v87 ida=0x8b62ed
// packet-audit:verify packet=messenger/serverbound/MessengerOperationAnswerInvite version=gms_v95 ida=0x7f59d0
// packet-audit:verify packet=messenger/serverbound/MessengerOperationAnswerInvite version=jms_v185 ida=0x8e11b0
func TestOperationAnswerInviteRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationAnswerInvite{messengerId: 42}
			output := OperationAnswerInvite{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.MessengerId() != input.MessengerId() {
				t.Errorf("messengerId: got %v, want %v", output.MessengerId(), input.MessengerId())
			}
		})
	}
}
