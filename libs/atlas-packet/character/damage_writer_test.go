package character

import (
	"testing"

	"github.com/Chronicle20/atlas-packet/model"
	"github.com/Chronicle20/atlas-packet/test"
)

func TestCharacterDamagePhysical(t *testing.T) {
	input := NewCharacterDamage(1234, model.DamageTypePhysical, 500, 100100, true)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
