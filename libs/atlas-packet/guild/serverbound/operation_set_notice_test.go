package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// CField::SendSetGuildNoticeMsg: COutPacket(GUILD_OPERATION)+Encode1(0x10=SET_NOTICE)+EncodeStr(notice).
// Body = EncodeStr(notice). IDA-verified: v83@0x530fa9, v84@0x53d228, v87@0x5587c9.
// v48 CField::SendSetGuildNoticeMsg @0x4c63d8 (sub_4C63D8): COutPacket(96=GUILD_OPERATION)+Encode1(0x10=SET_NOTICE)+EncodeStr(notice). Body=EncodeStr(notice), == v83.
// packet-audit:verify packet=guild/serverbound/GuildSetNotice version=gms_v48 ida=0x4c63d8
// packet-audit:verify packet=guild/serverbound/GuildSetNotice version=gms_v79 ida=0x51c59c
// v72 CField::SendSetGuildNoticeMsg @0x5154fd: COutPacket(124)+Encode1(0x10=SET_NOTICE)
// +EncodeStr(notice). Body = EncodeStr(notice), == v79.
// packet-audit:verify packet=guild/serverbound/GuildSetNotice version=gms_v72 ida=0x5154fd
// packet-audit:verify packet=guild/serverbound/GuildSetNotice version=gms_v95 ida=0x535180
// packet-audit:verify packet=guild/serverbound/GuildSetNotice version=jms_v185 ida=0x56e3a2
// packet-audit:verify packet=guild/serverbound/GuildSetNotice version=gms_v87 ida=0x5587c9
// packet-audit:verify packet=guild/serverbound/GuildSetNotice version=gms_v83 ida=0x530fa9
// packet-audit:verify packet=guild/serverbound/GuildSetNotice version=gms_v84 ida=0x53d228
// v61 COutPacket(114)+Encode1(16=SET_NOTICE)+EncodeStr(notice); body=EncodeStr, == v72/v83 (CField::SendSetGuildNoticeMsg @0x4e9b89).
// packet-audit:verify packet=guild/serverbound/GuildSetNotice version=gms_v61 ida=0x4e9b89
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
