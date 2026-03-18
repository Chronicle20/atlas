package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestOperationPersonalStoreSetBlackListRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationPersonalStoreSetBlackList{entries: []byte{1, 2, 3}}
			output := OperationPersonalStoreSetBlackList{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if len(output.Entries()) != len(input.Entries()) {
				t.Fatalf("entries length: got %v, want %v", len(output.Entries()), len(input.Entries()))
			}
			for i := range input.Entries() {
				if output.Entries()[i] != input.Entries()[i] {
					t.Errorf("entries[%d]: got %v, want %v", i, output.Entries()[i], input.Entries()[i])
				}
			}
		})
	}
}
