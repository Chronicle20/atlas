package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// CField::InputGuildName: COutPacket(GUILD_OPERATION)+Encode1(2=REQUEST_CREATE)+EncodeStr(name).
// Body after op+subtype = EncodeStr(name). IDA-verified: v83@0x5305ae, v84@0x53c812, v87@0x557db3.
// v48 CField::InputGuildName @0x4c5965: COutPacket(96=GUILD_OPERATION)+Encode1(2=REQUEST_CREATE)+EncodeStr(name). Body=EncodeStr(name), == v83.
// packet-audit:verify packet=guild/serverbound/GuildRequestCreate version=gms_v48 ida=0x4c5965
// packet-audit:verify packet=guild/serverbound/GuildRequestCreate version=gms_v79 ida=0x51bba1
// packet-audit:verify packet=guild/serverbound/GuildRequestCreate version=jms_v185 ida=0x56d98c
// packet-audit:verify packet=guild/serverbound/GuildRequestCreate version=gms_v95 ida=0x5347d0
// packet-audit:verify packet=guild/serverbound/GuildRequestCreate version=gms_v83 ida=0x5305ae
// packet-audit:verify packet=guild/serverbound/GuildRequestCreate version=gms_v84 ida=0x53c812
// packet-audit:verify packet=guild/serverbound/GuildRequestCreate version=gms_v87 ida=0x557db3
func TestRequestCreateRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := RequestCreate{name: "TestGuild"}
			output := RequestCreate{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Name() != input.Name() {
				t.Errorf("name: got %v, want %v", output.Name(), input.Name())
			}
		})
	}
}
