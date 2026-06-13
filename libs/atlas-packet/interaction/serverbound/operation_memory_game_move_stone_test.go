package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameMoveStone version=gms_v95 ida=0x6801e0
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameMoveStone version=gms_v87 ida=0x726570
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameMoveStone version=gms_v83 ida=0x6e8a19
func TestOperationMemoryGameMoveStoneRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationMemoryGameMoveStone{point: 123456789, color: 5}
			output := OperationMemoryGameMoveStone{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Point() != input.Point() {
				t.Errorf("point: got %v, want %v", output.Point(), input.Point())
			}
			if output.Color() != input.Color() {
				t.Errorf("color: got %v, want %v", output.Color(), input.Color())
			}
		})
	}
}
