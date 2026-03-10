package monster

import (
	"testing"

	"github.com/Chronicle20/atlas-packet/model"
	"github.com/Chronicle20/atlas-packet/test"
)

func TestMonsterStatSet(t *testing.T) {
	stat := model.NewMonsterTemporaryStat()
	input := NewMonsterStatSet(5001, stat)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestMonsterStatReset(t *testing.T) {
	stat := model.NewMonsterTemporaryStat()
	input := NewMonsterStatReset(5001, stat)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
