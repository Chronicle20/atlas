package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// CField::SendSetGuildNoticeMsg: COutPacket(GUILD_OPERATION)+Encode1(0x10=SET_NOTICE)+EncodeStr(notice).
// Body = EncodeStr(notice). IDA-verified: v83@0x530fa9, v84@0x53d228, v87@0x5587c9.
// packet-audit:verify packet=guild/serverbound/GuildSetNotice version=gms_v95 ida=0x535180
// packet-audit:verify packet=guild/serverbound/GuildSetNotice version=jms_v185 ida=0x56e3a2
// packet-audit:verify packet=guild/serverbound/GuildSetNotice version=gms_v87 ida=0x5587c9
// packet-audit:verify packet=guild/serverbound/GuildSetNotice version=gms_v83 ida=0x0
// packet-audit:verify packet=guild/serverbound/GuildSetNotice version=gms_v84 ida=0x0
func TestSetNoticeRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := SetNotice{notice: "Welcome to our guild!"}
			output := SetNotice{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Notice() != input.Notice() {
				t.Errorf("notice: got %v, want %v", output.Notice(), input.Notice())
			}
		})
	}
}
