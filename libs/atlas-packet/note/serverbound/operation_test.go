package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=note/serverbound/NoteOperation version=gms_v95 ida=0x9f3830
// packet-audit:verify packet=note/serverbound/NoteOperation version=gms_v87 ida=0xabce3b
// packet-audit:verify packet=note/serverbound/NoteOperation version=gms_v83 ida=0xa251ef
// packet-audit:verify packet=note/serverbound/NoteOperation version=jms_v185 ida=0xb0c849
// packet-audit:verify packet=note/serverbound/NoteOperation version=gms_v84 ida=0xa708ea
func TestOperationRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := Operation{op: 2}
			output := Operation{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Op() != input.Op() {
				t.Errorf("op: got %v, want %v", output.Op(), input.Op())
			}
		})
	}
}
