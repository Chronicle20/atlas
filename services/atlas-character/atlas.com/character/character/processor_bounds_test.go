package character

import (
	"math"
	"testing"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
)

// TestAppliesAutoAP verifies the GMS pre-Big-Bang auto-AP version boundary.
// v83 and v84 must both be included; v95+ excluded; non-GMS regions excluded.
func TestAppliesAutoAP(t *testing.T) {
	cases := []struct {
		region string
		major  uint16
		want   bool
	}{
		{"GMS", 83, true},  // v83 unchanged
		{"GMS", 84, true},  // v84 now included (was the bug)
		{"GMS", 94, true},  // pre-Big-Bang upper edge
		{"GMS", 95, false}, // post-Big-Bang excluded
		{"JMS", 83, false}, // region-gated
	}
	for _, c := range cases {
		tm, _ := tenant.Create(uuid.New(), c.region, c.major, 1)
		if got := appliesAutoAP(tm); got != c.want {
			t.Errorf("appliesAutoAP(%s,%d) = %v, want %v", c.region, c.major, got, c.want)
		}
	}
}

// TestEnforceBounds_DoesNotOverflowOnLargeCurrent pins the regression
// where enforceBounds did the intermediate sum in int16 — a character
// at HP=30000 receiving +5000 wrapped past int16 max into negative
// space and clamped to 0 (DIED). The fix uses int32 arithmetic.
func TestEnforceBounds_DoesNotOverflowOnLargeCurrent(t *testing.T) {
	got := enforceBounds(5000, 30000, 30000, 0)
	if got != 30000 {
		t.Fatalf("enforceBounds(+5000, 30000, max=30000, min=0) = %d, want 30000 (clamped to upper)", got)
	}
}

func TestEnforceBounds_AddWithinRange(t *testing.T) {
	got := enforceBounds(500, 1000, 5000, 0)
	if got != 1500 {
		t.Fatalf("enforceBounds(+500, 1000, ...) = %d, want 1500", got)
	}
}

func TestEnforceBounds_SubtractWithinRange(t *testing.T) {
	got := enforceBounds(-500, 1000, 5000, 0)
	if got != 500 {
		t.Fatalf("enforceBounds(-500, 1000, ...) = %d, want 500", got)
	}
}

func TestEnforceBounds_ClampsToLowerBound(t *testing.T) {
	got := enforceBounds(-1000, 200, 5000, 0)
	if got != 0 {
		t.Fatalf("enforceBounds(-1000, 200, min=0) = %d, want 0", got)
	}
}

func TestEnforceBounds_ClampsToUpperBound(t *testing.T) {
	got := enforceBounds(2000, 4000, 5000, 0)
	if got != 5000 {
		t.Fatalf("enforceBounds(+2000, 4000, max=5000) = %d, want 5000", got)
	}
}

// TestEnforceBounds_SaturatedNegativeChange exercises the same
// arithmetic-overflow risk in the opposite direction: current near
// uint16 max with a near-int16-min change. The int32 sum keeps math
// honest.
func TestEnforceBounds_SaturatedNegativeChange(t *testing.T) {
	got := enforceBounds(math.MinInt16, 60000, 65535, 0)
	if got != 60000-32768 {
		t.Fatalf("enforceBounds(MinInt16, 60000, ...) = %d, want %d", got, 60000-32768)
	}
}
