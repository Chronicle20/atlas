package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=messenger/serverbound/MessengerOperationDeclineInvite version=gms_v95 ida=0x524180
// packet-audit:verify packet=messenger/serverbound/MessengerOperationDeclineInvite version=jms_v185 ida=0x557267
// packet-audit:verify packet=messenger/serverbound/MessengerOperationDeclineInvite version=gms_v87 ida=0x54574f
// packet-audit:verify packet=messenger/serverbound/MessengerOperationDeclineInvite version=gms_v83 ida=0x51fff5
//
// v79 (CFadeWnd::SendCloseMessage @0x50be2e): the messenger window-type-0 arm
// (else branch) emits COutPacket(119=MESSENGER sb op) + Encode1(5)=mode +
// EncodeStr(fromName) + EncodeStr(myName) + Encode1(0). The op byte (119) and
// the leading mode byte (5) are stripped by the handler/dispatch; the body
// Atlas decodes = fromName + myName + alwaysZero(0), matching this codec. The
// guild-deny arms (case 6 op141 / case 8 op124) are the DENY_GUILD_REQUEST
// siblings. Body codec carries no MajorVersion gate (== v83).
// packet-audit:verify packet=messenger/serverbound/MessengerOperationDeclineInvite version=gms_v79 ida=0x50be2e
func TestOperationDeclineInviteRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationDeclineInvite{fromName: "Sender", myName: "Receiver", alwaysZero: 0}
			output := OperationDeclineInvite{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.FromName() != input.FromName() {
				t.Errorf("fromName: got %v, want %v", output.FromName(), input.FromName())
			}
			if output.MyName() != input.MyName() {
				t.Errorf("myName: got %v, want %v", output.MyName(), input.MyName())
			}
			if output.AlwaysZero() != input.AlwaysZero() {
				t.Errorf("alwaysZero: got %v, want %v", output.AlwaysZero(), input.AlwaysZero())
			}
		})
	}
}
