package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v48 CUIGuildBBS::SendViewEntryRequest @0x609211 (sub_609211): COutPacket(109=BBS_OPERATION)+Encode1(3=DISPLAY)+Encode4(threadId). Body=Encode4(threadId), == v83.
// packet-audit:verify packet=guild/serverbound/GuildBBSDisplayThread version=gms_v48 ida=0x609211
// packet-audit:verify packet=guild/serverbound/GuildBBSDisplayThread version=gms_v79 ida=0x786cdc
// v72 CUIGuildBBS::SendViewEntryRequest @0x751c9e: COutPacket(153)+Encode1(3)+Encode4(threadId), == v79.
// packet-audit:verify packet=guild/serverbound/GuildBBSDisplayThread version=gms_v72 ida=0x751c9e
// packet-audit:verify packet=guild/serverbound/GuildBBSDisplayThread version=gms_v87 ida=0x87a5df
// packet-audit:verify packet=guild/serverbound/GuildBBSDisplayThread version=gms_v95 ida=0x7c3710
// packet-audit:verify packet=guild/serverbound/GuildBBSDisplayThread version=gms_v83 ida=0x816bca
// v84 SendViewEntryRequest COutPacket(0x9F)+Encode1(3)+Encode4(threadId), IDA-verified.
// packet-audit:verify packet=guild/serverbound/GuildBBSDisplayThread version=gms_v84 ida=0x841e61
// packet-audit:verify packet=guild/serverbound/GuildBBSDisplayThread version=jms_v185 ida=ABSENT
// v61 COutPacket(134)+Encode1(3=DISPLAY)+Encode4(threadId); body=Encode4, == v72/v83 (CUIGuildBBS::SendViewEntryRequest @0x6bb5f9).
// packet-audit:verify packet=guild/serverbound/GuildBBSDisplayThread version=gms_v61 ida=0x6bb5f9
func TestBBSDisplayThreadRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := BBSDisplayThread{threadId: 15}
			output := BBSDisplayThread{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.ThreadId() != input.ThreadId() {
				t.Errorf("threadId: got %v, want %v", output.ThreadId(), input.ThreadId())
			}
		})
	}
}
