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

// identityEnvContext is a no-op envContext for tests that don't care about
// environment origination -- it just returns ctx unchanged.
func identityEnvContext(ctx context.Context) context.Context { return ctx }

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
	}, identityEnvContext)
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
	}, identityEnvContext)
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
	}, identityEnvContext)
	if n != 0 || called {
		t.Errorf("empty sweep must emit nothing: n=%d called=%v", n, called)
	}
}

// envMarkerKey is a test-local context key -- deliberately not
// libs/atlas-env, since character/combo sits outside env-domain-guard's
// permitted import list (main.go, kafka/, rest/, socket/) and must not
// import atlas-env even from a test file.
type envMarkerKey string

// TestProcessExpiriesAppliesEnvContextToCancel pins the review fix
// (task-32-review.md Major): processExpiries must run each expired combo's
// per-character context through envContext before calling cancel, so this
// per-character lifecycle event carries the pod's own environment identity
// rather than an empty one. Without this the DecayTick sweep would fail
// FR-1.8's decide() open, and the cancel would be actioned by every live
// deployment, not just the originating one.
func TestProcessExpiriesAppliesEnvContextToCancel(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	tn := testTenant(t)
	expired := []Expired{{t: tn, characterId: 1, f: testField(), comboId: skill.AranStage1ComboAbilityId}}

	envContext := func(ctx context.Context) context.Context {
		return context.WithValue(ctx, envMarkerKey("marker"), "stamped")
	}

	var gotMarker any
	n := processExpiries(l, context.Background(), expired, func(_ logrus.FieldLogger, ctx context.Context, _ Expired) error {
		gotMarker = ctx.Value(envMarkerKey("marker"))
		return nil
	}, envContext)

	if n != 1 {
		t.Fatalf("want 1 cancel, got %d", n)
	}
	if gotMarker != "stamped" {
		t.Fatalf("envContext was not applied to the cancel context: got %v, want \"stamped\"", gotMarker)
	}
}

func TestDecayTickSleepTime(t *testing.T) {
	tick := NewDecayTick(logrus.New(), context.Background(), time.Second, identityEnvContext)
	if got := tick.SleepTime(); got != time.Second {
		t.Errorf("want 1s, got %v", got)
	}
}
