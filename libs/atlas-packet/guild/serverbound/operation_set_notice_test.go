package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=guild/serverbound/GuildSetNotice version=gms_v95 ida=0x535180
// packet-audit:verify packet=guild/serverbound/GuildSetNotice version=jms_v185 ida=0x56e3a2
func TestSetNoticeRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := SetNotice{notice: "Welcome to our guild!"}
			output := SetNotice{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Notice() != input.Notice() {
				t.Errorf("notice: got %v, want %v", output.Notice(), input.Notice())
			}
		})
	}
}
