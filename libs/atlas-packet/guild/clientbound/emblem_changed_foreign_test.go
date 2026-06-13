package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=guild/clientbound/GuildForeignEmblemChanged version=jms_v185 ida=0xa57689
func TestForeignEmblemChangedRoundTrip(t *testing.T) {
	input := NewForeignEmblemChanged(1001, 3, 2, 5, 4)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := ForeignEmblemChanged{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}
