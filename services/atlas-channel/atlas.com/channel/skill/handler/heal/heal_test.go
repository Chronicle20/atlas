package heal

import (
	"testing"

	channelhandler "atlas-channel/skill/handler"

	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

func TestHeal_RegistersForClericHealId(t *testing.T) {
	h, ok := channelhandler.Lookup(skill2.ClericHealId)
	if !ok || h == nil {
		t.Fatalf("Lookup(ClericHealId) = (%v, %v), want non-nil handler", h, ok)
	}
}
