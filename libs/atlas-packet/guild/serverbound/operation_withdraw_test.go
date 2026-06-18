package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// CField::SendWithdrawGuildMsg: COutPacket(GUILD_OPERATION)+Encode1(7=WITHDRAW)+Encode4(cid)+EncodeStr(name).
// Body = Encode4(cid)+EncodeStr(name). IDA-verified: v83@0x5308e0, v84@0x53cb4d, v87@0x5580ee.
// packet-audit:verify packet=guild/serverbound/GuildWithdraw version=jms_v185 ida=0x56dcc7
// packet-audit:verify packet=guild/serverbound/GuildWithdraw version=gms_v95 ida=0x534ad0
// packet-audit:verify packet=guild/serverbound/GuildWithdraw version=gms_v83 ida=0x5308e0
// packet-audit:verify packet=guild/serverbound/GuildWithdraw version=gms_v84 ida=0x53cb4d
// packet-audit:verify packet=guild/serverbound/GuildWithdraw version=gms_v87 ida=0x5580ee
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
