package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestCharacterSkillCooldown(t *testing.T) {
	input := NewCharacterSkillCooldown(1001003, 30)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
