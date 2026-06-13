package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=guild/serverbound/GuildBBSDisplayThread version=gms_v87 ida=0x87a5df
// packet-audit:verify packet=guild/serverbound/GuildBBSDisplayThread version=gms_v95 ida=0x7c3710
// packet-audit:verify packet=guild/serverbound/GuildBBSDisplayThread version=gms_v83 ida=0x0
// packet-audit:verify packet=guild/serverbound/GuildBBSDisplayThread version=jms_v185 ida=ABSENT
func TestBBSDisplayThreadRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := BBSDisplayThread{threadId: 15}
			output := BBSDisplayThread{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.ThreadId() != input.ThreadId() {
				t.Errorf("threadId: got %v, want %v", output.ThreadId(), input.ThreadId())
			}
		})
	}
}
