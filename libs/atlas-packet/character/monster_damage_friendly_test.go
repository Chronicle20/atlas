package character

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestMonsterDamageFriendlyRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := MonsterDamageFriendly{attackerId: 100, observerId: 200, attackedId: 300}
			output := MonsterDamageFriendly{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.AttackerId() != input.AttackerId() {
				t.Errorf("attackerId: got %v, want %v", output.AttackerId(), input.AttackerId())
			}
			if output.ObserverId() != input.ObserverId() {
				t.Errorf("observerId: got %v, want %v", output.ObserverId(), input.ObserverId())
			}
			if output.AttackedId() != input.AttackedId() {
				t.Errorf("attackedId: got %v, want %v", output.AttackedId(), input.AttackedId())
			}
		})
	}
}
