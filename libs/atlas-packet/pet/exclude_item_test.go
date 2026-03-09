package pet

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestExcludeItemRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ExcludeItem{petId: 12345, itemIds: []int32{1000, 2000, 3000}}
			output := ExcludeItem{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.PetId() != input.PetId() {
				t.Errorf("petId: got %v, want %v", output.PetId(), input.PetId())
			}
			if len(output.ItemIds()) != len(input.ItemIds()) {
				t.Fatalf("itemIds length: got %v, want %v", len(output.ItemIds()), len(input.ItemIds()))
			}
			for i, id := range output.ItemIds() {
				if id != input.ItemIds()[i] {
					t.Errorf("itemIds[%d]: got %v, want %v", i, id, input.ItemIds()[i])
				}
			}
		})
	}
}
