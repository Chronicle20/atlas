package serverbound

import (
	"encoding/binary"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/sirupsen/logrus/hooks/test"
)

func TestActionRestoreLostItemCountArray(t *testing.T) {
	l, _ := test.NewNullLogger()
	in := ActionRestoreLostItem{itemIds: []uint32{4000001, 4000002}}
	b := in.Encode(l, pt.CreateContext("GMS", 83, 1))(nil)
	if len(b) != 12 {
		t.Fatalf("got %d bytes, want 12: % x", len(b), b)
	}
	if got := binary.LittleEndian.Uint32(b[0:4]); got != 2 {
		t.Errorf("count = %d, want 2", got)
	}
	if got := binary.LittleEndian.Uint32(b[4:8]); got != 4000001 {
		t.Errorf("itemIds[0] = %d, want 4000001", got)
	}
	if got := binary.LittleEndian.Uint32(b[8:12]); got != 4000002 {
		t.Errorf("itemIds[1] = %d, want 4000002", got)
	}
}

func TestActionRestoreLostItemRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ActionRestoreLostItem{itemIds: []uint32{4001000, 4000002, 4000003}}
			output := ActionRestoreLostItem{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if len(output.ItemIds()) != len(input.ItemIds()) {
				t.Fatalf("itemIds len: got %d, want %d", len(output.ItemIds()), len(input.ItemIds()))
			}
			for i := range input.ItemIds() {
				if output.ItemIds()[i] != input.ItemIds()[i] {
					t.Errorf("itemIds[%d]: got %v, want %v", i, output.ItemIds()[i], input.ItemIds()[i])
				}
			}
		})
	}
}
