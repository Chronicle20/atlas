package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=guild/serverbound/GuildBBSListThreads version=gms_v87 ida=0x87a5df
// packet-audit:verify packet=guild/serverbound/GuildBBSListThreads version=gms_v95 ida=0x7c3680
// packet-audit:verify packet=guild/serverbound/GuildBBSListThreads version=gms_v83 ida=0x0
// packet-audit:verify packet=guild/serverbound/GuildBBSListThreads version=jms_v185 ida=ABSENT
func TestBBSListThreadsRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := BBSListThreads{startIndex: 10}
			output := BBSListThreads{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.StartIndex() != input.StartIndex() {
				t.Errorf("startIndex: got %v, want %v", output.StartIndex(), input.StartIndex())
			}
		})
	}
}
