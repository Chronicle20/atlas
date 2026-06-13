package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=guild/clientbound/GuildForeignNameChanged version=jms_v185 ida=0xa5763e
// packet-audit:verify packet=guild/clientbound/GuildForeignNameChanged version=gms_v95 ida=0x0
func TestForeignNameChangedRoundTrip(t *testing.T) {
	input := NewForeignNameChanged(1001, "NewGuildName")
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := ForeignNameChanged{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}
