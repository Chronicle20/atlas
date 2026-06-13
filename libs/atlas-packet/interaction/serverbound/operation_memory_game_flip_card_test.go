package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameFlipCard version=gms_v95 ida=0x6279b0
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameFlipCard version=gms_v87 ida=0x688d3b
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameFlipCard version=gms_v83 ida=0x64ee2b
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameFlipCard version=jms_v185 ida=0x6c8b94
func TestOperationMemoryGameFlipCardRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationMemoryGameFlipCard{first: true, index: 7}
			output := OperationMemoryGameFlipCard{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.First() != input.First() {
				t.Errorf("first: got %v, want %v", output.First(), input.First())
			}
			if output.Index() != input.Index() {
				t.Errorf("index: got %v, want %v", output.Index(), input.Index())
			}
		})
	}
}
