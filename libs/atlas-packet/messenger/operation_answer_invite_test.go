package messenger

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestOperationAnswerInviteRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationAnswerInvite{messengerId: 42}
			output := OperationAnswerInvite{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.MessengerId() != input.MessengerId() {
				t.Errorf("messengerId: got %v, want %v", output.MessengerId(), input.MessengerId())
			}
		})
	}
}
