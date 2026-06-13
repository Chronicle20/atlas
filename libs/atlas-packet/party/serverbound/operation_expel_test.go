package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=party/serverbound/PartyOperationExpel version=gms_v95 ida=0x530140
// packet-audit:verify packet=party/serverbound/PartyOperationExpel version=jms_v185 ida=0x56cf23
func TestOperationExpelRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationExpel{targetCharacterId: 200}
			output := OperationExpel{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.TargetCharacterId() != input.TargetCharacterId() {
				t.Errorf("targetCharacterId: got %v, want %v", output.TargetCharacterId(), input.TargetCharacterId())
			}
		})
	}
}
