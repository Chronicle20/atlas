package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestCommandRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := Command{petId: 12345, byName: true, command: 3}
			output := Command{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.PetId() != input.PetId() {
				t.Errorf("petId: got %v, want %v", output.PetId(), input.PetId())
			}
			if output.ByName() != input.ByName() {
				t.Errorf("byName: got %v, want %v", output.ByName(), input.ByName())
			}
			if output.Command() != input.Command() {
				t.Errorf("command: got %v, want %v", output.Command(), input.Command())
			}
		})
	}
}
