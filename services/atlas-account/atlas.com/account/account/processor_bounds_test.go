package account

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
)

func TestUsesChooseGender(t *testing.T) {
	cases := []struct {
		region string
		major  uint16
		want   bool
	}{
		{"GMS", 83, false}, // v83 -> Male (unchanged)
		{"GMS", 84, false}, // v84 -> Male (the off-by-one fix; was UI-choose)
		{"GMS", 86, false}, // still pre-v87 -> Male
		{"GMS", 87, true},  // v87+ -> UI-choose (unchanged from >83)
		{"GMS", 95, true},  // UI-choose
		{"JMS", 87, false}, // region-gated
	}
	for _, c := range cases {
		tm, _ := tenant.Create(uuid.New(), c.region, c.major, 1)
		if got := usesChooseGender(tm); got != c.want {
			t.Errorf("usesChooseGender(%s,%d) = %v, want %v", c.region, c.major, got, c.want)
		}
	}
}
