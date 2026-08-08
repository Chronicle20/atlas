package poisonmist

import (
	"atlas-channel/data/skill/effect"
	"context"
	"errors"
	"strings"
	"testing"

	mistmsg "atlas-channel/kafka/message/mist"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

const (
	testCharId  = uint32(1001)
	testSkillId = uint32(2111003)
	testLevel   = byte(1)
	testX       = int16(500)
	testY       = int16(300)
)

// stubEffect builds an effect.Model carrying the level-1 Poison Mist values
// read from the provisioned WZ corpus (task-200 design §2.1): time 4s (4000ms
// after the reader's conversion), lt (-110,-82), rb (110,83).
//
// effect.Model has unexported fields and no exported constructor; hydrating
// through effect.Extract on a RestModel literal is the supported construction
// path.
func stubEffect(durationMs int32, ltX, ltY, rbX, rbY int16) effect.Model {
	m, err := effect.Extract(effect.RestModel{
		Duration: durationMs,
		LT:       &effect.PointRestModel{X: ltX, Y: ltY},
		RB:       &effect.PointRestModel{X: rbX, Y: rbY},
	})
	if err != nil {
		panic(err)
	}
	return m
}

// harness swaps both package seams and restores them on cleanup, returning a
// pointer to the slice of emitted bodies.
func harness(t *testing.T, casterErr error) *[]mistmsg.CreateCommandBody {
	t.Helper()
	origLoad, origEmit := loadCaster, emitCreate
	t.Cleanup(func() { loadCaster, emitCreate = origLoad, origEmit })

	emitted := make([]mistmsg.CreateCommandBody, 0)
	loadCaster = func(logrus.FieldLogger, context.Context, uint32) (int16, int16, error) {
		if casterErr != nil {
			return 0, 0, casterErr
		}
		return testX, testY, nil
	}
	emitCreate = func(_ logrus.FieldLogger, _ context.Context, body mistmsg.CreateCommandBody) error {
		emitted = append(emitted, body)
		return nil
	}
	return &emitted
}

func testField() field.Model {
	return field.NewBuilder(0, 0, 100000000).SetInstance(uuid.Nil).Build()
}

func run(t *testing.T, e effect.Model) (*[]mistmsg.CreateCommandBody, *test.Hook) {
	t.Helper()
	l, hook := test.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)
	emitted := harness(t, nil)
	// The WIRE skill id is passed, not the resolved Identity: 2111003 on all
	// eleven provisioned versions, and the handler forwards it verbatim
	// because the client compares it against its own WZ.
	err := Apply(l)(context.Background())(nil, testField(), testCharId, skill2.Id(testSkillId), testLevel, e)
	require.NoError(t, err)
	return emitted, hook
}

// TestApply_HappyPath_EmitsExactlyOneCreate pins every field of the emitted
// CREATE body against the task-200 design §4.2 table.
func TestApply_HappyPath_EmitsExactlyOneCreate(t *testing.T) {
	emitted, _ := run(t, stubEffect(4000, -110, -82, 110, 83))

	require.Len(t, *emitted, 1)
	b := (*emitted)[0]
	require.Equal(t, "CHARACTER", b.OwnerType)
	require.Equal(t, testCharId, b.OwnerId)
	require.Equal(t, mistmsg.TargetKindMonster, b.TargetKind)
	require.Equal(t, mistmsg.EffectKindDamageOverTime, b.EffectKind)
	require.Equal(t, testX, b.OriginX)
	require.Equal(t, testY, b.OriginY)
	require.Equal(t, int16(-110), b.LtX)
	require.Equal(t, int16(-82), b.LtY)
	require.Equal(t, int16(110), b.RbX)
	require.Equal(t, int16(83), b.RbY)
	require.Equal(t, "POISON", b.Disease)
	require.Equal(t, int32(0), b.DiseaseValue)       // design D1c -- magnitude unread for POISON
	require.Equal(t, int64(4000), b.DiseaseDuration) // design D1a -- per-target = mist lifetime
	require.Equal(t, int64(4000), b.Duration)
	require.Equal(t, PlayerMistTickIntervalMs, b.TickIntervalMs)
	require.Equal(t, testSkillId, b.SourceSkillId) // WIRE id -- the client compares it
	require.Equal(t, uint32(testLevel), b.SourceSkillLevel)
}

// TestApply_ZeroLifetime_Rejected covers FR-6.1.
func TestApply_ZeroLifetime_Rejected(t *testing.T) {
	emitted, hook := run(t, stubEffect(0, -110, -82, 110, 83))
	require.Empty(t, *emitted)
	requireLogged(t, hook, "no lifetime")
}

// TestApply_LifetimeShorterThanOneTick_Rejected covers FR-6.2: a mist that
// expires before its first re-apply tick is an invisible no-op. Pinned at the
// gate's actual boundary (PlayerMistTickIntervalMs - 1ms), not an arbitrary
// small value, so a future change to the constant can't silently widen or
// narrow what this test proves.
func TestApply_LifetimeShorterThanOneTick_Rejected(t *testing.T) {
	emitted, hook := run(t, stubEffect(int32(PlayerMistTickIntervalMs)-1, -110, -82, 110, 83))
	require.Empty(t, *emitted)
	requireLogged(t, hook, "lifetime shorter than one tick")
}

// TestApply_LifetimeEqualToOneTick_Accepted pins the other side of the FR-6.2
// boundary: a duration exactly equal to PlayerMistTickIntervalMs must NOT be
// rejected. This also covers the shortest legitimate Poison Mist cast --
// level 1's 4000ms lifetime (task-200 design §2.1 table) -- which must always
// clear this gate.
func TestApply_LifetimeEqualToOneTick_Accepted(t *testing.T) {
	emitted, _ := run(t, stubEffect(int32(PlayerMistTickIntervalMs), -110, -82, 110, 83))
	require.Len(t, *emitted, 1)
}

// TestApply_DegenerateRectangle_Rejected covers FR-6.3.
func TestApply_DegenerateRectangle_Rejected(t *testing.T) {
	emitted, hook := run(t, stubEffect(4000, 110, -82, -110, 83))
	require.Empty(t, *emitted)
	requireLogged(t, hook, "degenerate rectangle")
}

// TestApply_ImplausibleLifetime_Rejected covers FR-6.4. The largest legitimate
// `time` for 2111003 is 40s at level 30, so 5 minutes can only be corrupt data.
func TestApply_ImplausibleLifetime_Rejected(t *testing.T) {
	emitted, hook := run(t, stubEffect(MaxPlayerMistDurationMs+1, -110, -82, 110, 83))
	require.Empty(t, *emitted)
	requireLogged(t, hook, "implausible lifetime")
}

// TestApply_CasterLoadFailure_EmitsNothingAndReturnsNil covers FR-3.3: no
// mist, and no error surfaced to the client.
func TestApply_CasterLoadFailure_EmitsNothingAndReturnsNil(t *testing.T) {
	l, _ := test.NewNullLogger()
	emitted := harness(t, errors.New("character service down"))
	err := Apply(l)(context.Background())(nil, testField(), testCharId, skill2.Id(testSkillId), testLevel, stubEffect(4000, -110, -82, 110, 83))
	require.NoError(t, err)
	require.Empty(t, *emitted)
}

// requireLogged asserts one log entry contains the given rejection reason, so
// each FR-6 gate is distinguishable in production (NFR-4).
func requireLogged(t *testing.T, hook *test.Hook, want string) {
	t.Helper()
	for _, e := range hook.AllEntries() {
		if strings.Contains(e.Message, want) {
			return
		}
	}
	t.Fatalf("no log entry containing %q; got %v", want, hook.AllEntries())
}
