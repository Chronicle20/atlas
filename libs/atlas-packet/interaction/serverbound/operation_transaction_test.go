package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=interaction/serverbound/InteractionOperationTransaction version=gms_v95 ida=0x49e180
// packet-audit:verify packet=interaction/serverbound/InteractionOperationTransaction version=gms_v87 ida=0x494dcb
// packet-audit:verify packet=interaction/serverbound/InteractionOperationTransaction version=gms_v83 ida=0x485dcd
// packet-audit:verify packet=interaction/serverbound/InteractionOperationTransaction version=jms_v185 ida=0x499b67
func TestOperationTransactionRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationTransaction{entries: []TransactionEntry{{data: 100, crc: 200}, {data: 300, crc: 400}}}
			output := OperationTransaction{}
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
