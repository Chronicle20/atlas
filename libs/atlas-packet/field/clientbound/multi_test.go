package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldMultiChat version=gms_v83 ida=0x531e00
// packet-audit:verify packet=field/clientbound/FieldMultiChat version=gms_v84 ida=0x53e086
// packet-audit:verify packet=field/clientbound/FieldMultiChat version=gms_v87 ida=0x5596b1
// packet-audit:verify packet=field/clientbound/FieldMultiChat version=gms_v95 ida=0x535490
// packet-audit:verify packet=field/clientbound/FieldMultiChat version=jms_v185 ida=0x56f286
func TestMultiChatRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := MultiChat{mode: 1, from: "PlayerOne", message: "party chat message"}
			output := MultiChat{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.From() != input.From() {
				t.Errorf("from: got %v, want %v", output.From(), input.From())
			}
			if output.Message() != input.Message() {
				t.Errorf("message: got %v, want %v", output.Message(), input.Message())
			}
		})
	}
}
