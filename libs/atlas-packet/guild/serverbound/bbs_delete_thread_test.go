package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=guild/serverbound/GuildBBSDeleteThread version=gms_v79 ida=0x7869ed
// v72 CUIGuildBBS::OnDelete @0x7519af: COutPacket(153)+Encode1(1)+Encode4(threadId), == v79.
// packet-audit:verify packet=guild/serverbound/GuildBBSDeleteThread version=gms_v72 ida=0x7519af
// packet-audit:verify packet=guild/serverbound/GuildBBSDeleteThread version=gms_v87 ida=0x87a5df
// packet-audit:verify packet=guild/serverbound/GuildBBSDeleteThread version=gms_v95 ida=0x7c6520
// packet-audit:verify packet=guild/serverbound/GuildBBSDeleteThread version=gms_v83 ida=0x8168db
// v84 OnDelete COutPacket(0x9F)+Encode1(1)+Encode4(threadId), IDA-verified.
// packet-audit:verify packet=guild/serverbound/GuildBBSDeleteThread version=gms_v84 ida=0x841b72
// packet-audit:verify packet=guild/serverbound/GuildBBSDeleteThread version=jms_v185 ida=ABSENT
// v61 COutPacket(134)+Encode1(1=DELETE)+Encode4(threadId); body=Encode4, == v72/v83 (CUIGuildBBS::OnDelete @0x6bb30c).
// packet-audit:verify packet=guild/serverbound/GuildBBSDeleteThread version=gms_v61 ida=0x6bb30c
func TestBBSDeleteThreadRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := BBSDeleteThread{threadId: 7}
			output := BBSDeleteThread{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.ThreadId() != input.ThreadId() {
				t.Errorf("threadId: got %v, want %v", output.ThreadId(), input.ThreadId())
			}
		})
	}
}
