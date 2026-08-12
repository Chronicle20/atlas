package combo

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

func TestProcessExpiriesCancelsOncePerEntry(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	tn := testTenant(t)
	expired := []Expired{
		{t: tn, characterId: 1, f: testField(), comboId: skill.AranStage1ComboAbilityId},
		{t: tn, characterId: 2, f: testField(), comboId: skill.LegendComboAbilityId},
	}
	var seen []uint32
	n := processExpiries(l, context.Background(), expired, func(_ logrus.FieldLogger, _ context.Context, e Expired) error {
		seen = append(seen, e.CharacterId())
		return nil
	})
	if n != 2 || len(seen) != 2 {
		t.Fatalf("want 2 cancels, got n=%d seen=%v", n, seen)
	}
}

func TestProcessExpiriesSwallowsCancelFailure(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	tn := testTenant(t)
	expired := []Expired{{t: tn, characterId: 1, f: testField(), comboId: skill.AranStage1ComboAbilityId}}
	n := processExpiries(l, context.Background(), expired, func(_ logrus.FieldLogger, _ context.Context, _ Expired) error {
		return errors.New("broker down")
	})
	if n != 0 {
		t.Errorf("failed cancels are not counted as processed: want 0, got %d", n)
	}
}

func TestProcessExpiriesEmptySweepDoesNothing(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	called := false
	n := processExpiries(l, context.Background(), nil, func(_ logrus.FieldLogger, _ context.Context, _ Expired) error {
		called = true
		return nil
	})
	if n != 0 || called {
		t.Errorf("empty sweep must emit nothing: n=%d called=%v", n, called)
	}
}

func TestDecayTickSleepTime(t *testing.T) {
	tick := NewDecayTick(logrus.New(), context.Background(), time.Second)
	if got := tick.SleepTime(); got != time.Second {
		t.Errorf("want 1s, got %v", got)
	}
}
