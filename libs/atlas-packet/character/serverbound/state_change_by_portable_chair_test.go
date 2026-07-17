package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/serverbound/StateChangeByPortableChair version=gms_v83 ida=0xa02e34
// packet-audit:verify packet=character/serverbound/StateChangeByPortableChair version=gms_v84 ida=0xa4d05a
// packet-audit:verify packet=character/serverbound/StateChangeByPortableChair version=gms_v87 ida=0xa97e50
// packet-audit:verify packet=character/serverbound/StateChangeByPortableChair version=gms_v95 ida=0x9d4020
// packet-audit:verify packet=character/serverbound/StateChangeByPortableChair version=jms_v185 ida=0xae6f5a
func TestStateChangeByPortableChairRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := StateChangeByPortableChair{}

			// The body is empty in every version: encode must emit zero bytes.
			b := pt.Encode(t, ctx, input.Encode, nil)
			if len(b) != 0 {
				t.Errorf("body: got %d bytes, want 0", len(b))
			}

			// Decode must consume nothing (RoundTrip asserts 0 unconsumed).
			output := StateChangeByPortableChair{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}
