package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas-packet/test"
)

func TestMonsterDamage(t *testing.T) {
	input := NewMonsterDamage(5001, MonsterDamageTypeUnk2, 1500, 8500, 10000)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
