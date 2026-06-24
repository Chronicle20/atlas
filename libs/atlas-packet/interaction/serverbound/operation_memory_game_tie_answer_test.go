package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameTieAnswer version=gms_v95 ida=0x627e60
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameTieAnswer version=gms_v87 ida=0x68826d
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameTieAnswer version=gms_v83 ida=0x64e363
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameTieAnswer version=jms_v185 ida=0x6c815e
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameTieAnswer version=gms_v84 ida=0x664034
func TestOperationMemoryGameTieAnswerRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationMemoryGameTieAnswer{response: true}
			output := OperationMemoryGameTieAnswer{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Response() != input.Response() {
				t.Errorf("response: got %v, want %v", output.Response(), input.Response())
			}
		})
	}
}
