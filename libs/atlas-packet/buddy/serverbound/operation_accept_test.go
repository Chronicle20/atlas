package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestOperationAcceptRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationAccept{fromCharacterId: 12345}
			output := OperationAccept{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.FromCharacterId() != input.FromCharacterId() {
				t.Errorf("fromCharacterId: got %v, want %v", output.FromCharacterId(), input.FromCharacterId())
			}
		})
	}
}
