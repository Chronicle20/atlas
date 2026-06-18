package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=guild/serverbound/GuildBBSReplyThread version=gms_v87 ida=0x87a5df
// packet-audit:verify packet=guild/serverbound/GuildBBSReplyThread version=gms_v95 ida=0x7c4530
// packet-audit:verify packet=guild/serverbound/GuildBBSReplyThread version=gms_v83 ida=0x816994
// v84 OnComment COutPacket(0x9F)+Encode1(4)+Encode4(threadId)+EncodeStr(message), IDA-verified.
// packet-audit:verify packet=guild/serverbound/GuildBBSReplyThread version=gms_v84 ida=0x841c2b
// packet-audit:verify packet=guild/serverbound/GuildBBSReplyThread version=jms_v185 ida=ABSENT
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
