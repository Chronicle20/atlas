package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v48 CUIGuildBBS::SendLoadListRequest @0x6091b1 (sub_6091B1): COutPacket(109=BBS_OPERATION)+Encode1(2=LIST)+Encode4(startIndex). Body=Encode4(startIndex), == v83.
// packet-audit:verify packet=guild/serverbound/GuildBBSListThreads version=gms_v48 ida=0x6091b1
// packet-audit:verify packet=guild/serverbound/GuildBBSListThreads version=gms_v79 ida=0x786c7b
// v72 CUIGuildBBS::SendLoadListRequest @0x751c3d: COutPacket(153=BBS_OPERATION)+Encode1(2)+Encode4(startIndex). Body=Encode4(startIndex), == v79.
// packet-audit:verify packet=guild/serverbound/GuildBBSListThreads version=gms_v72 ida=0x751c3d
// packet-audit:verify packet=guild/serverbound/GuildBBSListThreads version=gms_v87 ida=0x87a5df
// packet-audit:verify packet=guild/serverbound/GuildBBSListThreads version=gms_v95 ida=0x7c3680
// packet-audit:verify packet=guild/serverbound/GuildBBSListThreads version=gms_v83 ida=0x816b69
// v84 BBS opcode 0x9F (NOT v83's 0x9B): SendLoadListRequest COutPacket(159)+Encode1(2)+Encode4(startIndex), IDA-verified.
// packet-audit:verify packet=guild/serverbound/GuildBBSListThreads version=gms_v84 ida=0x841e00
// packet-audit:verify packet=guild/serverbound/GuildBBSListThreads version=jms_v185 ida=ABSENT
// v61 COutPacket(134)+Encode1(2=LIST)+Encode4(startIndex); body=Encode4, == v72/v83 (CUIGuildBBS::SendLoadListRequest @0x6bb596).
// packet-audit:verify packet=guild/serverbound/GuildBBSListThreads version=gms_v61 ida=0x6bb596
func TestBBSListThreadsRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := BBSListThreads{startIndex: 10}
			output := BBSListThreads{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.StartIndex() != input.StartIndex() {
				t.Errorf("startIndex: got %v, want %v", output.StartIndex(), input.StartIndex())
			}
		})
	}
}
