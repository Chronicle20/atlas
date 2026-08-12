package periodic

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
)

// TestLookupRows pins every row's shape. The values are WZ-verified in
// docs/tasks/task-214-buff-tick-effects/design.md §2 — a row edited by
// accident fails here rather than in production.
func TestLookupRows(t *testing.T) {
	tests := []struct {
		statType  character.TemporaryStatType
		interval  time.Duration
		resource  Resource
		direction Direction
		floor     bool
	}{
		{character.TemporaryStatTypePoison, time.Second, ResourceHP, Drain, false},
		{character.TemporaryStatTypeDragonBlood, 4 * time.Second, ResourceHP, Drain, true},
		{character.TemporaryStatTypeRecovery, 5 * time.Second, ResourceHP, Restore, false},
	}

	for _, tc := range tests {
		t.Run(string(tc.statType), func(t *testing.T) {
			e, ok := Lookup(string(tc.statType))
			assert.True(t, ok, "expected a table row")
			assert.Equal(t, tc.statType, e.StatType())
			assert.Equal(t, tc.interval, e.Interval())
			assert.Equal(t, tc.resource, e.Resource())
			assert.Equal(t, tc.direction, e.Direction())
			assert.Equal(t, tc.floor, e.Floor())
		})
	}
}

// TestLookupRowCount fails when a row is added without a matching assertion in
// TestLookupRows, so the table cannot grow silently.
func TestLookupRowCount(t *testing.T) {
	assert.Len(t, effects, 3)
}

// TestLookupNonPeriodic covers stat types atlas-data emits that design.md §5.3
// gives an "excluded" verdict: the tick path must not pick them up.
func TestLookupNonPeriodic(t *testing.T) {
	for _, st := range []character.TemporaryStatType{
		character.TemporaryStatTypeInfinity,
		character.TemporaryStatTypeMagicGuard,
		character.TemporaryStatTypeWeaponAttack,
		character.TemporaryStatTypeHolyShield,
	} {
		_, ok := Lookup(string(st))
		assert.False(t, ok, "unexpected periodic row for %s", st)
	}
}

func TestLookupUnknownStatType(t *testing.T) {
	_, ok := Lookup("NOT_A_REAL_STAT")
	assert.False(t, ok)
}

func TestDirectionSigns(t *testing.T) {
	assert.Equal(t, Direction(-1), Drain)
	assert.Equal(t, Direction(1), Restore)
}
