package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestHealOverTimeRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := HealOverTime{updateTime: 100, val: 200, hp: 50, mp: 30, unknown: 1}
			output := HealOverTime{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.HP() != input.HP() {
				t.Errorf("hp: got %v, want %v", output.HP(), input.HP())
			}
			if output.MP() != input.MP() {
				t.Errorf("mp: got %v, want %v", output.MP(), input.MP())
			}
			if output.UpdateTime() != input.UpdateTime() {
				t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
			}
		})
	}
}
