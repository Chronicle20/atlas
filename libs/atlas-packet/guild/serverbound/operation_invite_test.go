package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// CField::SendInviteGuildMsg: COutPacket(GUILD_OPERATION)+Encode1(5=INVITE)+EncodeStr(target).
// Body after op+subtype = EncodeStr(target). IDA-verified: v83@0x5306d5, v84@0x53c93f, v87@0x557ee0.
// v48 CField::SendInviteGuildMsg @0x4c5a89: COutPacket(96=GUILD_OPERATION)+Encode1(5=INVITE)+EncodeStr(target). Body=EncodeStr(target), == v83.
// packet-audit:verify packet=guild/serverbound/GuildInviteRequest version=gms_v48 ida=0x4c5a89
// packet-audit:verify packet=guild/serverbound/GuildInviteRequest version=gms_v79 ida=0x51bcc8
// packet-audit:verify packet=guild/serverbound/GuildInviteRequest version=jms_v185 ida=0x56dab9
// packet-audit:verify packet=guild/serverbound/GuildInviteRequest version=gms_v95 ida=0x5348e0
// packet-audit:verify packet=guild/serverbound/GuildInviteRequest version=gms_v83 ida=0x5306d5
// packet-audit:verify packet=guild/serverbound/GuildInviteRequest version=gms_v84 ida=0x53c93f
// packet-audit:verify packet=guild/serverbound/GuildInviteRequest version=gms_v87 ida=0x557ee0
func TestInviteRequestRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := InviteRequest{target: "InvitedPlayer"}
			output := InviteRequest{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Target() != input.Target() {
				t.Errorf("target: got %v, want %v", output.Target(), input.Target())
			}
		})
	}
}
