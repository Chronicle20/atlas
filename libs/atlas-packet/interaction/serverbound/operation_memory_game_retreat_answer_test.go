package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameRetreatAnswer version=gms_v95 ida=0x6804b0
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameRetreatAnswer version=gms_v87 ida=0x721c80
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameRetreatAnswer version=gms_v83 ida=0x6e416b
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameRetreatAnswer version=jms_v185 ida=0x72b69c
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameRetreatAnswer version=gms_v84 ida=0x6fb416
func TestOperationMemoryGameRetreatAnswerRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationMemoryGameRetreatAnswer{response: true}
			output := OperationMemoryGameRetreatAnswer{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Response() != input.Response() {
				t.Errorf("response: got %v, want %v", output.Response(), input.Response())
			}
		})
	}
}
