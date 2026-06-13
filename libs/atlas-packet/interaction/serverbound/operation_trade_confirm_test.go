package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=interaction/serverbound/InteractionOperationTradeConfirm version=gms_v95 ida=0x7646b0
// packet-audit:verify packet=interaction/serverbound/InteractionOperationTradeConfirm version=gms_v87 ida=0x8170d5
// packet-audit:verify packet=interaction/serverbound/InteractionOperationTradeConfirm version=gms_v83 ida=0x7c39a0
// packet-audit:verify packet=interaction/serverbound/InteractionOperationTradeConfirm version=jms_v185 ida=0x84830a
// packet-audit:verify packet=interaction/serverbound/InteractionOperationTradeConfirm version=gms_v84 ida=0x7e9ae6
func TestOperationTradeConfirmRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationTradeConfirm{entries: []TradeConfirmEntry{{data: 100, crc: 200}, {data: 300, crc: 400}}}
			output := OperationTradeConfirm{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if len(output.Entries()) != len(input.Entries()) {
				t.Fatalf("entries length: got %v, want %v", len(output.Entries()), len(input.Entries()))
			}
			for i := range input.Entries() {
				if output.Entries()[i].Data() != input.Entries()[i].Data() {
					t.Errorf("entries[%d].data: got %v, want %v", i, output.Entries()[i].Data(), input.Entries()[i].Data())
				}
				if output.Entries()[i].Crc() != input.Entries()[i].Crc() {
					t.Errorf("entries[%d].crc: got %v, want %v", i, output.Entries()[i].Crc(), input.Entries()[i].Crc())
				}
			}
		})
	}
}
