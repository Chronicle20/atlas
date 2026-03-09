package buddy

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestOperationDeleteRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationDelete{buddyCharacterId: 67890}
			output := OperationDelete{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.BuddyCharacterId() != input.BuddyCharacterId() {
				t.Errorf("buddyCharacterId: got %v, want %v", output.BuddyCharacterId(), input.BuddyCharacterId())
			}
		})
	}
}
