package monster

import (
	"atlas-monsters/monster/information"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// dotTickTestSelfDestruction is the Boomer's WZ selfDestruction block used
// across these tests: {action: 1, removeAfter: -1, hp: 1800}.
func dotTickTestSelfDestruction() information.SelfDestruction {
	return information.NewSelfDestruction(true, 1, -1, 1800)
}

// setupDoTTickTest installs a capturing producer manager (restored to the
// TestMain no-op floor on cleanup) and swaps testInformationLookup to return
// sd for every monster template lookup. Returns the monster registry's
// tenant-scoped context and the capture to assert emitted events against.
func setupDoTTickTest(t *testing.T, sd information.SelfDestruction) (context.Context, tenant.Model, *producertest.Capture) {
	t.Helper()
	r := GetMonsterRegistry()
	ten := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), ten)
	r.Clear(ctx)

	capture := producertest.InstallCapturing()
	t.Cleanup(producertest.InstallNoop)

	prevHook := testInformationLookup
	testInformationLookup = func(_ uint32) (information.Model, error) {
		return information.NewModelBuilder().SetSelfDestruction(sd).Build(), nil
	}
	t.Cleanup(func() { testInformationLookup = prevHook })

	return ctx, ten, capture
}

// killedEvents decodes every EventMonsterStatusKilled body captured on the
// monster status topic.
func killedEvents(t *testing.T, capture *producertest.Capture) []statusEventKilledBody {
	t.Helper()
	var bodies []statusEventKilledBody
	for _, msg := range capture.Messages(EnvEventTopicMonsterStatus) {
		var env struct {
			Type string          `json:"type"`
			Body json.RawMessage `json:"body"`
		}
		require.NoError(t, json.Unmarshal(msg.Value, &env))
		if env.Type != EventMonsterStatusKilled {
			continue
		}
		var body statusEventKilledBody
		require.NoError(t, json.Unmarshal(env.Body, &body))
		bodies = append(bodies, body)
	}
	return bodies
}

// TestDoTTickCrossingThresholdDetonates: a Boomer poisoned for 50/tick at HP
// 1830 crosses its 1800 threshold on the tick that lands it at 1780 -- the
// monster detonates instead of merely taking damage.
func TestDoTTickCrossingThresholdDetonates(t *testing.T) {
	ctx, ten, capture := setupDoTTickTest(t, dotTickTestSelfDestruction())

	m := GetMonsterRegistry().CreateMonster(ctx, ten, testField(), boomerMonsterId, 0, 0, 0, 5, 0, 1830, 0, "", "")
	se := NewStatusEffect(SourceTypePlayerSkill, 88, 2111003, 30,
		map[string]int32{StatusPoison: 50}, 40*time.Second, time.Second)

	task := &StatusExpirationTask{l: logrus.New()}
	task.processDoTTick(ten, ctx, m, se)

	_, err := GetMonsterRegistry().GetMonster(ten, m.UniqueId())
	require.Error(t, err, "monster must be absent from the registry after detonation")

	bodies := killedEvents(t, capture)
	require.Len(t, bodies, 1, "expected exactly one KILLED event")
	require.Equal(t, byte(1), bodies[0].DeathType)
	require.Equal(t, uint32(88), bodies[0].ActorId)
}

// TestDoTTickNotCrossingThresholdDoesNotDetonate: the same tick from a
// higher starting HP lands above the threshold and must not detonate.
func TestDoTTickNotCrossingThresholdDoesNotDetonate(t *testing.T) {
	ctx, ten, capture := setupDoTTickTest(t, dotTickTestSelfDestruction())

	m := GetMonsterRegistry().CreateMonster(ctx, ten, testField(), boomerMonsterId, 0, 0, 0, 5, 0, 4000, 0, "", "")
	se := NewStatusEffect(SourceTypePlayerSkill, 88, 2111003, 30,
		map[string]int32{StatusPoison: 50}, 40*time.Second, time.Second)

	task := &StatusExpirationTask{l: logrus.New()}
	task.processDoTTick(ten, ctx, m, se)

	updated, err := GetMonsterRegistry().GetMonster(ten, m.UniqueId())
	require.NoError(t, err)
	require.Equal(t, uint32(3950), updated.Hp())

	require.Empty(t, killedEvents(t, capture))
}

// TestDoTTickCannotReachZeroHp is a regression on the existing kill-prevention
// cap: a mob with no selfDestruction block must still never be reduced to 0
// HP by poison, regardless of the threshold logic added here.
func TestDoTTickCannotReachZeroHp(t *testing.T) {
	ctx, ten, capture := setupDoTTickTest(t, information.NewSelfDestruction(false, 0, -1, -1))

	m := GetMonsterRegistry().CreateMonster(ctx, ten, testField(), boomerMonsterId, 0, 0, 0, 5, 0, 30, 0, "", "")
	se := NewStatusEffect(SourceTypePlayerSkill, 88, 2111003, 30,
		map[string]int32{StatusPoison: 500}, 40*time.Second, time.Second)

	task := &StatusExpirationTask{l: logrus.New()}
	task.processDoTTick(ten, ctx, m, se)

	updated, err := GetMonsterRegistry().GetMonster(ten, m.UniqueId())
	require.NoError(t, err)
	require.Equal(t, uint32(1), updated.Hp())
	require.True(t, updated.Alive())

	require.Empty(t, killedEvents(t, capture))
}

// TestDoTTickThresholdMobStillCannotBeReducedToZero: the kill-prevention cap
// clamps the tick to currentHp-1 (1809 here), landing at HP 1 -- which is
// still <= the 1800 threshold, so the mob detonates from that clamped tick.
func TestDoTTickThresholdMobStillCannotBeReducedToZero(t *testing.T) {
	ctx, ten, capture := setupDoTTickTest(t, dotTickTestSelfDestruction())

	m := GetMonsterRegistry().CreateMonster(ctx, ten, testField(), boomerMonsterId, 0, 0, 0, 5, 0, 1810, 0, "", "")
	se := NewStatusEffect(SourceTypePlayerSkill, 88, 2111003, 30,
		map[string]int32{StatusPoison: 5000}, 40*time.Second, time.Second)

	task := &StatusExpirationTask{l: logrus.New()}
	task.processDoTTick(ten, ctx, m, se)

	_, err := GetMonsterRegistry().GetMonster(ten, m.UniqueId())
	require.Error(t, err, "monster must be absent from the registry after detonation")

	bodies := killedEvents(t, capture)
	require.Len(t, bodies, 1, "expected exactly one KILLED event")
	require.Equal(t, byte(1), bodies[0].DeathType)
}
