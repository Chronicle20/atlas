package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=storage/serverbound/StorageOperationMeso version=gms_v95 ida=0x7688e0
// packet-audit:verify packet=storage/serverbound/StorageOperationMeso version=gms_v87 ida=0x81c15c
// packet-audit:verify packet=storage/serverbound/StorageOperationMeso version=jms_v185 ida=0x84e3dc
func TestOperationMesoRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationMeso{amount: -5000}
			output := OperationMeso{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Amount() != input.Amount() {
				t.Errorf("amount: got %v, want %v", output.Amount(), input.Amount())
			}
		})
	}
}
