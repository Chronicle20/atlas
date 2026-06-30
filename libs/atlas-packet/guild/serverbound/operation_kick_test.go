package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// CField::SendKickGuildMsg: COutPacket(GUILD_OPERATION)+Encode1(8=KICK)+Encode4(cid)+EncodeStr(name).
// Body = Encode4(cid)+EncodeStr(name). IDA-verified: v83@0x530a0d, v84@0x53cc7d, v87@0x55821e.
// packet-audit:verify packet=guild/serverbound/GuildKick version=gms_v79 ida=0x51c000
// packet-audit:verify packet=guild/serverbound/GuildKick version=jms_v185 ida=0x56ddf7
// packet-audit:verify packet=guild/serverbound/GuildKick version=gms_v95 ida=0x534cb0
// packet-audit:verify packet=guild/serverbound/GuildKick version=gms_v83 ida=0x530a0d
// packet-audit:verify packet=guild/serverbound/GuildKick version=gms_v84 ida=0x53cc7d
// packet-audit:verify packet=guild/serverbound/GuildKick version=gms_v87 ida=0x55821e
func TestKickRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := Kick{cid: 67890, name: "BadPlayer"}
			output := Kick{}
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
