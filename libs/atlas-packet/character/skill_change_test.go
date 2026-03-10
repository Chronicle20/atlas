package character

import (
	"testing"
	"time"

	"github.com/Chronicle20/atlas-packet/test"
)

func TestCharacterSkillChange(t *testing.T) {
	input := NewCharacterSkillChange(true, 1001003, 10, 0, time.Time{}, false)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
