package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// CField::SendWithdrawGuildMsg: COutPacket(GUILD_OPERATION)+Encode1(7=WITHDRAW)+Encode4(cid)+EncodeStr(name).
// Body = Encode4(cid)+EncodeStr(name). IDA-verified: v83@0x5308e0, v84@0x53cb4d, v87@0x5580ee.
// v48 CField::SendWithdrawGuildMsg @0x4c5cc4 (sub_4C5CC4, YesNo==6 arm): COutPacket(96=GUILD_OPERATION)+Encode1(7=WITHDRAW)+Encode4(cid)+EncodeStr(GetCharacterName). Body=Encode4(cid)+EncodeStr(name), == v83.
// packet-audit:verify packet=guild/serverbound/GuildWithdraw version=gms_v48 ida=0x4c5cc4
// packet-audit:verify packet=guild/serverbound/GuildWithdraw version=gms_v79 ida=0x51bed3
// v72 CField::SendWithdrawGuildMsg @0x514e34 (YesNo==6 arm): COutPacket(124=GUILD_OPERATION)
// +Encode1(7=WITHDRAW)+Encode4(cid)+EncodeStr(GetCharacterName). Body = Encode4(cid)+EncodeStr(name), == v79.
// packet-audit:verify packet=guild/serverbound/GuildWithdraw version=gms_v72 ida=0x514e34
// packet-audit:verify packet=guild/serverbound/GuildWithdraw version=jms_v185 ida=0x56dcc7
// packet-audit:verify packet=guild/serverbound/GuildWithdraw version=gms_v95 ida=0x534ad0
// packet-audit:verify packet=guild/serverbound/GuildWithdraw version=gms_v83 ida=0x5308e0
// packet-audit:verify packet=guild/serverbound/GuildWithdraw version=gms_v84 ida=0x53cb4d
// packet-audit:verify packet=guild/serverbound/GuildWithdraw version=gms_v87 ida=0x5580ee
// v61 COutPacket(114)+Encode1(7=WITHDRAW)+Encode4(cid)+EncodeStr(name); body=Encode4+EncodeStr, == v72/v83 (CField::SendWithdrawGuildMsg @0x4e94cf).
// packet-audit:verify packet=guild/serverbound/GuildWithdraw version=gms_v61 ida=0x4e94cf
func TestWithdrawRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := Withdraw{cid: 12345, name: "SomePlayer"}
			output := Withdraw{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Cid() != input.Cid() {
				t.Errorf("cid: got %v, want %v", output.Cid(), input.Cid())
			}
			if output.Name() != input.Name() {
				t.Errorf("name: got %v, want %v", output.Name(), input.Name())
			}
		})
	}
}
