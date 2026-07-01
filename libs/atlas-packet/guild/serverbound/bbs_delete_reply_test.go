package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=guild/serverbound/GuildBBSDeleteReply version=gms_v79 ida=0x786bba
// v72 CUIGuildBBS::OnCommentDelete @0x751b7c: COutPacket(153)+Encode1(5)+Encode4(threadId)+Encode4(replyId), == v79.
// packet-audit:verify packet=guild/serverbound/GuildBBSDeleteReply version=gms_v72 ida=0x751b7c
// packet-audit:verify packet=guild/serverbound/GuildBBSDeleteReply version=gms_v87 ida=0x87a5df
// packet-audit:verify packet=guild/serverbound/GuildBBSDeleteReply version=gms_v95 ida=0x7c3b70
// packet-audit:verify packet=guild/serverbound/GuildBBSDeleteReply version=gms_v83 ida=0x816aa8
// v84 OnCommentDelete COutPacket(0x9F)+Encode1(5)+Encode4(threadId)+Encode4(replyId), IDA-verified.
// packet-audit:verify packet=guild/serverbound/GuildBBSDeleteReply version=gms_v84 ida=0x841d3f
// packet-audit:verify packet=guild/serverbound/GuildBBSDeleteReply version=jms_v185 ida=ABSENT
func TestBBSDeleteReplyRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := BBSDeleteReply{threadId: 3, replyId: 12}
			output := BBSDeleteReply{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.ThreadId() != input.ThreadId() {
				t.Errorf("threadId: got %v, want %v", output.ThreadId(), input.ThreadId())
			}
			if output.ReplyId() != input.ReplyId() {
				t.Errorf("replyId: got %v, want %v", output.ReplyId(), input.ReplyId())
			}
		})
	}
}
