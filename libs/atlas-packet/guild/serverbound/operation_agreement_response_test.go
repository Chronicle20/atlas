package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestAgreementResponseRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := AgreementResponse{unk: 42, agreed: true}
			output := AgreementResponse{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Unk() != input.Unk() {
				t.Errorf("unk: got %v, want %v", output.Unk(), input.Unk())
			}
			if output.Agreed() != input.Agreed() {
				t.Errorf("agreed: got %v, want %v", output.Agreed(), input.Agreed())
			}
		})
	}
}
