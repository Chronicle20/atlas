package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/serverbound/ItemCancel version=gms_v83 ida=0xa096af
// packet-audit:verify packet=character/serverbound/ItemCancel version=gms_v87 ida=0xa9ef5b
// packet-audit:verify packet=character/serverbound/ItemCancel version=gms_v95 ida=0x9d9dd0
func TestItemCancelRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ItemCancel{sourceId: 2001001}
			output := ItemCancel{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.SourceId() != input.SourceId() {
				t.Errorf("sourceId: got %v, want %v", output.SourceId(), input.SourceId())
			}
		})
	}
}
