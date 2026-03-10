package messenger

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestMessengerRemoveRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewMessengerRemove(2, 1)
			output := RemoveW{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.Position() != input.Position() {
				t.Errorf("position: got %v, want %v", output.Position(), input.Position())
			}
		})
	}
}
