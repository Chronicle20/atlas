package character

import (
	"errors"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

// Regression: atlas-character was emitting spurious DIED events during routine
// HP regen because ChangeHP unconditionally trusted the value returned by the
// effective-stats service, even when that value was zero. A positive amount +
// upperBound=0 in enforceBounds clamped HP to 0, which triggered
// diedEventProvider with killerType=UNKNOWN.
//
// The helper below is the fix's decision gate: error OR zero → fall back to
// the character's base max; positive → use effective.
func TestResolveEffectiveMax(t *testing.T) {
	tests := []struct {
		name       string
		base       uint16
		effective  uint32
		fetchErr   error
		want       uint16
		wantLogMsg string
	}{
		{
			name:      "effective stats positive uses effective",
			base:      50,
			effective: 337,
			fetchErr:  nil,
			want:      337,
		},
		{
			name:       "effective stats returned zero falls back to base and warns",
			base:       337,
			effective:  0,
			fetchErr:   nil,
			want:       337,
			wantLogMsg: "reported MaxHP=0",
		},
		{
			name:       "effective stats fetch error falls back to base, debug only",
			base:       337,
			effective:  0,
			fetchErr:   errors.New("connection refused"),
			want:       337,
			wantLogMsg: "Failed to fetch effective stats",
		},
		{
			name:      "effective stats reports lower value uses that",
			base:      337,
			effective: 200,
			fetchErr:  nil,
			want:      200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l, hook := test.NewNullLogger()
			l.SetLevel(logrus.DebugLevel)

			got := resolveEffectiveMax(l, tt.base, tt.effective, tt.fetchErr, 12, "MaxHP")
			if got != tt.want {
				t.Errorf("resolveEffectiveMax = %d, want %d", got, tt.want)
			}

			if tt.wantLogMsg != "" {
				found := false
				for _, e := range hook.AllEntries() {
					if containsSubstring(e.Message, tt.wantLogMsg) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected log message containing %q; entries: %+v", tt.wantLogMsg, hook.AllEntries())
				}
			}
		})
	}
}

// The critical regression scenario: positive HP regen + zero-valued effective
// stats must NOT clamp HP to zero. This composes resolveEffectiveMax with
// enforceBounds exactly as ChangeHP does.
func TestRegenWithZeroEffectiveMaxPreservesHP(t *testing.T) {
	l, _ := test.NewNullLogger()

	currentHP := uint16(337)
	baseMaxHP := uint16(337)
	regenDelta := int16(10) // what character_heal_over_time.go:20 sends

	// stats service succeeds but reports zero (the observed bug)
	maxHP := resolveEffectiveMax(l, baseMaxHP, 0, nil, 12, "MaxHP")
	adjusted := enforceBounds(regenDelta, currentHP, maxHP, 0)

	if adjusted == 0 {
		t.Fatalf("HP was clamped to 0 on +%d regen — regression, character would die spuriously", regenDelta)
	}
	if adjusted != baseMaxHP {
		t.Errorf("HP after +%d regen = %d, want %d (clamped at base max)", regenDelta, adjusted, baseMaxHP)
	}
}

func containsSubstring(haystack, needle string) bool {
	if needle == "" {
		return true
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
