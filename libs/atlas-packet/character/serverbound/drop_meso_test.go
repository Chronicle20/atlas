package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/serverbound/DropMeso version=gms_v83 ida=0xa23de5
// packet-audit:verify packet=character/serverbound/DropMeso version=gms_v87 ida=0xabb8b3
// packet-audit:verify packet=character/serverbound/DropMeso version=gms_v95 ida=0x9f6650
func TestDropMesoRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := DropMeso{updateTime: 100, amount: 5000}
			output := DropMeso{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Amount() != input.Amount() {
				t.Errorf("amount: got %v, want %v", output.Amount(), input.Amount())
			}
		})
	}
}
