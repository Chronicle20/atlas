package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=guild/serverbound/GuildOperation version=gms_v79 ida=0x50e36f
// v72 CUIFadeYesNo::OnButtonClicked @0x50285e: the GUILD_OPERATION serverbound
// dispatcher base (button-click cases emit COutPacket(124)+Encode1(mode)+body);
// the generic op envelope (op+mode) is == v79.
// packet-audit:verify packet=guild/serverbound/GuildOperation version=gms_v72 ida=0x50285e
// packet-audit:verify packet=guild/serverbound/GuildOperation version=gms_v83 ida=0x522585
// packet-audit:verify packet=guild/serverbound/GuildOperation version=gms_v84 ida=0x52dc20
// packet-audit:verify packet=guild/serverbound/GuildOperation version=gms_v87 ida=0x548098
// packet-audit:verify packet=guild/serverbound/GuildOperation version=gms_v95 ida=0x529c60
// packet-audit:verify packet=guild/serverbound/GuildOperation version=jms_v185 ida=0x5599d6
func TestOperationRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := Operation{op: 7}
			output := Operation{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Op() != input.Op() {
				t.Errorf("op: got %v, want %v", output.Op(), input.Op())
			}
		})
	}
}
