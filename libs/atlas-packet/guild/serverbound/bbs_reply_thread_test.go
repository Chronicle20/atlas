package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v48 CUIGuildBBS::OnComment @0x608fe6 (sub_608FE6): COutPacket(109=BBS_OPERATION)+Encode1(4=REPLY)+Encode4(threadId)+EncodeStr(message). Body=Encode4(threadId)+EncodeStr(message), == v83.
// packet-audit:verify packet=guild/serverbound/GuildBBSReplyThread version=gms_v48 ida=0x608fe6
// packet-audit:verify packet=guild/serverbound/GuildBBSReplyThread version=gms_v79 ida=0x786aa6
// v72 CUIGuildBBS::OnComment @0x751a68: COutPacket(153)+Encode1(4)+Encode4(threadId)+EncodeStr(message), == v79.
// packet-audit:verify packet=guild/serverbound/GuildBBSReplyThread version=gms_v72 ida=0x751a68
// packet-audit:verify packet=guild/serverbound/GuildBBSReplyThread version=gms_v87 ida=0x87a5df
// packet-audit:verify packet=guild/serverbound/GuildBBSReplyThread version=gms_v95 ida=0x7c4530
// packet-audit:verify packet=guild/serverbound/GuildBBSReplyThread version=gms_v83 ida=0x816994
// v84 OnComment COutPacket(0x9F)+Encode1(4)+Encode4(threadId)+EncodeStr(message), IDA-verified.
// packet-audit:verify packet=guild/serverbound/GuildBBSReplyThread version=gms_v84 ida=0x841c2b
// packet-audit:verify packet=guild/serverbound/GuildBBSReplyThread version=jms_v185 ida=ABSENT
// v61 COutPacket(134)+Encode1(4=REPLY)+Encode4(threadId)+EncodeStr(message); body=Encode4+EncodeStr, == v72/v83 (CUIGuildBBS::OnComment @0x6bb3c4).
// packet-audit:verify packet=guild/serverbound/GuildBBSReplyThread version=gms_v61 ida=0x6bb3c4
func TestBBSReplyThreadRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := BBSReplyThread{threadId: 9, message: "Nice post!"}
			output := BBSReplyThread{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.ThreadId() != input.ThreadId() {
				t.Errorf("threadId: got %v, want %v", output.ThreadId(), input.ThreadId())
			}
			if output.Message() != input.Message() {
				t.Errorf("message: got %v, want %v", output.Message(), input.Message())
			}
		})
	}
}
