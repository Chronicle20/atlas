package character

import (
	"testing"

	"github.com/Chronicle20/atlas-packet/model"
	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestBuffCancelWRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			cts := model.NewCharacterTemporaryStat()
			input := NewBuffCancelW(*cts)
			output := BuffCancelW{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

func TestBuffCancelForeignRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			cts := model.NewCharacterTemporaryStat()
			input := NewBuffCancelForeign(99999, *cts)
			output := BuffCancelForeign{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.CharacterId() != 99999 {
				t.Errorf("characterId: got %v, want %v", output.CharacterId(), 99999)
			}
		})
	}
}
