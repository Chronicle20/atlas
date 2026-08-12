package character

import (
	"atlas-buffs/buff/stat"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	character2 "atlas-buffs/kafka/message/character"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// counterValue reads a labeled counter from the default gatherer (0 when the
// series does not exist yet). Same shape as atlas-login's
// character/processor_test.go helper.
func counterValue(t *testing.T, name, labelName, labelValue string) float64 {
	t.Helper()
	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	for _, mf := range mfs {
		if mf.GetName() != name {
			continue
		}
		for _, m := range mf.GetMetric() {
			for _, lp := range m.GetLabel() {
				if lp.GetName() == labelName && lp.GetValue() == labelValue {
					return m.GetCounter().GetValue()
				}
			}
		}
	}
	return 0
}

// tickProcessor builds a ProcessorImpl with a frozen clock and a stubbed HP
// read. Same-package struct literal, mirroring berserk/processor_test.go's
// testProcessor — no test-helper file, no test-only constructor.
func tickProcessor(ctx context.Context, now *time.Time, hp uint16, hpErr error, hpCalls *int) *ProcessorImpl {
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		now: func() time.Time { return *now },
		getCharacterHp: func(_ uint32) (uint16, error) {
			*hpCalls++
			return hp, hpErr
		},
	}
}

// changeHPAmounts decodes every CHANGE_HP command captured so far, in order.
func changeHPAmounts(t *testing.T) []int16 {
	t.Helper()
	var out []int16
	for _, m := range emitted.Messages(character2.EnvCommandTopicCharacter) {
		var cmd character2.CharacterCommand[character2.ChangeHPCommandBody]
		require.NoError(t, json.Unmarshal(m.Value, &cmd))
		if cmd.Type != character2.CommandChangeHP {
			continue
		}
		out = append(out, cmd.Body.Amount)
	}
	return out
}

func changeHPCommands(t *testing.T) []character2.CharacterCommand[character2.ChangeHPCommandBody] {
	t.Helper()
	var out []character2.CharacterCommand[character2.ChangeHPCommandBody]
	for _, m := range emitted.Messages(character2.EnvCommandTopicCharacter) {
		var cmd character2.CharacterCommand[character2.ChangeHPCommandBody]
		require.NoError(t, json.Unmarshal(m.Value, &cmd))
		out = append(out, cmd)
	}
	return out
}

// applyBuff stores a live buff. 600000 is MILLISECONDS (10 minutes) — long
// enough that no test's frozen-clock arithmetic can outlive it, since
// buff.Model.Expired() reads the real wall clock.
func applyBuff(t *testing.T, ctx context.Context, characterId uint32, sourceId int32, changes ...stat.Model) {
	t.Helper()
	_, err := GetRegistry().Apply(ctx, world.Id(0), channel.Id(1), characterId, sourceId, 1, 600000, changes, false, false)
	require.NoError(t, err)
}

// TestPeriodicTickPoisonParity pins the pre-task-214 poison behavior: a stored
// POISON amount of 25 emits CHANGE_HP -25, is suppressed 500ms later, and
// emits again at 1s (FR-2.4).
func TestPeriodicTickPoisonParity(t *testing.T) {
	setupTestRegistry(t)
	emitted.Reset()
	ctx := setupTestContext(t, setupTestTenant(t))
	now := time.Now()
	calls := 0
	p := tickProcessor(ctx, &now, 100, nil, &calls)

	applyBuff(t, ctx, 100, 2111003, stat.NewStat("POISON", 25))

	require.NoError(t, p.ProcessPeriodicTicks())
	assert.Equal(t, []int16{-25}, changeHPAmounts(t))

	now = now.Add(500 * time.Millisecond)
	require.NoError(t, p.ProcessPeriodicTicks())
	assert.Equal(t, []int16{-25}, changeHPAmounts(t), "throttled inside the 1s interval")

	now = now.Add(500 * time.Millisecond)
	require.NoError(t, p.ProcessPeriodicTicks())
	assert.Equal(t, []int16{-25, -25}, changeHPAmounts(t), "emits again at 1s")

	assert.Zero(t, calls, "POISON has no floor, so no HP read is made")
}

// TestPeriodicTickCommandShape pins the emitted command's envelope.
func TestPeriodicTickCommandShape(t *testing.T) {
	setupTestRegistry(t)
	emitted.Reset()
	ctx := setupTestContext(t, setupTestTenant(t))
	now := time.Now()
	calls := 0
	p := tickProcessor(ctx, &now, 100, nil, &calls)

	applyBuff(t, ctx, 100, 2111003, stat.NewStat("POISON", 25))
	require.NoError(t, p.ProcessPeriodicTicks())

	cmds := changeHPCommands(t)
	require.Len(t, cmds, 1)
	assert.Equal(t, uint32(100), cmds[0].CharacterId)
	assert.Equal(t, world.Id(0), cmds[0].WorldId)
	assert.Equal(t, character2.CommandChangeHP, cmds[0].Type)
	assert.Equal(t, channel.Id(1), cmds[0].Body.ChannelId)
	assert.Equal(t, int16(-25), cmds[0].Body.Amount)
}

// TestPeriodicTickSkipsNonPositiveMagnitude preserves the poison guard
// (`amount >= 0 { continue }`) generically (FR-1.5).
func TestPeriodicTickSkipsNonPositiveMagnitude(t *testing.T) {
	setupTestRegistry(t)
	emitted.Reset()
	ctx := setupTestContext(t, setupTestTenant(t))
	now := time.Now()
	calls := 0
	p := tickProcessor(ctx, &now, 100, nil, &calls)

	applyBuff(t, ctx, 100, 2111003, stat.NewStat("POISON", 0))
	applyBuff(t, ctx, 101, 2111003, stat.NewStat("POISON", -5))

	require.NoError(t, p.ProcessPeriodicTicks())
	assert.Empty(t, changeHPAmounts(t))
}

// TestPeriodicTickIndependentCadences: POISON (1s) and DRAGON_BLOOD (4s) on one
// character throttle independently (FR-2.2).
func TestPeriodicTickIndependentCadences(t *testing.T) {
	setupTestRegistry(t)
	emitted.Reset()
	ctx := setupTestContext(t, setupTestTenant(t))
	now := time.Now()
	calls := 0
	p := tickProcessor(ctx, &now, 500, nil, &calls)

	applyBuff(t, ctx, 100, 1311008,
		stat.NewStat("POISON", 25),
		stat.NewStat("DRAGON_BLOOD", 48))

	require.NoError(t, p.ProcessPeriodicTicks())
	assert.ElementsMatch(t, []int16{-25, -48}, changeHPAmounts(t), "t=0 emits both")

	emitted.Reset()
	now = now.Add(time.Second)
	require.NoError(t, p.ProcessPeriodicTicks())
	assert.Equal(t, []int16{-25}, changeHPAmounts(t), "t=1s: POISON only")

	emitted.Reset()
	now = now.Add(3 * time.Second)
	require.NoError(t, p.ProcessPeriodicTicks())
	assert.ElementsMatch(t, []int16{-25, -48}, changeHPAmounts(t), "t=4s: both")
}

// TestPeriodicTickDragonBloodFloorsAtOne (FR-3.4).
func TestPeriodicTickDragonBloodFloorsAtOne(t *testing.T) {
	tests := []struct {
		name string
		hp   uint16
		want []int16
	}{
		{"full hp drains the whole amount", 100, []int16{-48}},
		{"low hp is reduced so hp lands on 1", 30, []int16{-29}},
		{"exactly enough to land on 1", 49, []int16{-48}},
		{"at 1 hp emits nothing", 1, nil},
		{"at 0 hp emits nothing", 0, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setupTestRegistry(t)
			emitted.Reset()
			ctx := setupTestContext(t, setupTestTenant(t))
			now := time.Now()
			calls := 0
			p := tickProcessor(ctx, &now, tc.hp, nil, &calls)

			applyBuff(t, ctx, 100, 1311008, stat.NewStat("DRAGON_BLOOD", 48))
			require.NoError(t, p.ProcessPeriodicTicks())
			assert.Equal(t, tc.want, changeHPAmounts(t))
		})
	}
}

// TestPeriodicTickDragonBloodFailsClosedOnHpError (design D5). A failed HP
// read must also be observed via degrade.Observe — logged and counted —
// so a sustained CHARACTERS outage that silently disables the HP floor
// leaves a metric signal, not just a skipped tick.
func TestPeriodicTickDragonBloodFailsClosedOnHpError(t *testing.T) {
	setupTestRegistry(t)
	emitted.Reset()
	ctx := setupTestContext(t, setupTestTenant(t))
	now := time.Now()
	calls := 0
	p := tickProcessor(ctx, &now, 0, errors.New("boom"), &calls)

	before := counterValue(t, "atlas_enrichment_degraded_total", "component", "buffs.periodic.character_hp")

	applyBuff(t, ctx, 100, 1311008, stat.NewStat("DRAGON_BLOOD", 48))
	require.NoError(t, p.ProcessPeriodicTicks(), "a failed HP read is not a pass failure")
	assert.Empty(t, changeHPAmounts(t), "never emit an unclamped drain")

	after := counterValue(t, "atlas_enrichment_degraded_total", "component", "buffs.periodic.character_hp")
	assert.Equal(t, float64(1), after-before, "expected one degradation observation for the failed HP read")
}

// TestPeriodicTickHpReadIsMemoizedPerPass (FR-3.6): a character with two
// floor-sensitive rows reads HP once. Today DRAGON_BLOOD is the only floor row,
// so the bound is asserted as "at most one read per affected character per
// pass" across two passes.
func TestPeriodicTickHpReadIsMemoizedPerPass(t *testing.T) {
	setupTestRegistry(t)
	emitted.Reset()
	ctx := setupTestContext(t, setupTestTenant(t))
	now := time.Now()
	calls := 0
	p := tickProcessor(ctx, &now, 500, nil, &calls)

	applyBuff(t, ctx, 100, 1311008,
		stat.NewStat("DRAGON_BLOOD", 48),
		stat.NewStat("POISON", 25),
		stat.NewStat("RECOVERY", 4))

	require.NoError(t, p.ProcessPeriodicTicks())
	assert.Equal(t, 1, calls, "one HP read for the one floor-sensitive character")
}

// TestPeriodicTickRecoveryOnlyMakesNoHpRead (NFR load bound).
func TestPeriodicTickRecoveryOnlyMakesNoHpRead(t *testing.T) {
	setupTestRegistry(t)
	emitted.Reset()
	ctx := setupTestContext(t, setupTestTenant(t))
	now := time.Now()
	calls := 0
	p := tickProcessor(ctx, &now, 500, nil, &calls)

	applyBuff(t, ctx, 100, 10001001, stat.NewStat("RECOVERY", 4))
	require.NoError(t, p.ProcessPeriodicTicks())

	assert.Equal(t, []int16{4}, changeHPAmounts(t), "positive and unclamped by atlas-buffs (FR-4.4)")
	assert.Zero(t, calls)
}

// TestPeriodicTickRecoveryCadence: 5s, per WZ (FR-4.2).
func TestPeriodicTickRecoveryCadence(t *testing.T) {
	setupTestRegistry(t)
	emitted.Reset()
	ctx := setupTestContext(t, setupTestTenant(t))
	now := time.Now()
	calls := 0
	p := tickProcessor(ctx, &now, 500, nil, &calls)

	applyBuff(t, ctx, 100, 10001001, stat.NewStat("RECOVERY", 4))

	require.NoError(t, p.ProcessPeriodicTicks())
	now = now.Add(4 * time.Second)
	require.NoError(t, p.ProcessPeriodicTicks())
	assert.Equal(t, []int16{4}, changeHPAmounts(t), "throttled at 4s")

	now = now.Add(time.Second)
	require.NoError(t, p.ProcessPeriodicTicks())
	assert.Equal(t, []int16{4, 4}, changeHPAmounts(t), "emits at 5s")
}

// TestPeriodicTickDedupesDuplicatePoisonBuffs (design D7).
func TestPeriodicTickDedupesDuplicatePoisonBuffs(t *testing.T) {
	setupTestRegistry(t)
	emitted.Reset()
	ctx := setupTestContext(t, setupTestTenant(t))
	now := time.Now()
	calls := 0
	p := tickProcessor(ctx, &now, 500, nil, &calls)

	applyBuff(t, ctx, 100, 5001, stat.NewStat("POISON", 10))
	applyBuff(t, ctx, 100, 5002, stat.NewStat("POISON", 25))

	require.NoError(t, p.ProcessPeriodicTicks())
	assert.Equal(t, []int16{-25}, changeHPAmounts(t))
}

// TestPeriodicTickIsTenantScoped: same character id in two tenants ticks twice
// and throttles independently.
func TestPeriodicTickIsTenantScoped(t *testing.T) {
	setupTestRegistry(t)
	emitted.Reset()
	ctxA := setupTestContext(t, setupTestTenant(t))
	ctxB := setupTestContext(t, setupTestTenant(t))
	now := time.Now()
	callsA, callsB := 0, 0
	pA := tickProcessor(ctxA, &now, 500, nil, &callsA)
	pB := tickProcessor(ctxB, &now, 500, nil, &callsB)

	applyBuff(t, ctxA, 100, 2111003, stat.NewStat("POISON", 25))
	applyBuff(t, ctxB, 100, 2111003, stat.NewStat("POISON", 11))

	require.NoError(t, pA.ProcessPeriodicTicks())
	require.NoError(t, pB.ProcessPeriodicTicks())
	assert.ElementsMatch(t, []int16{-25, -11}, changeHPAmounts(t))
}

// TestPeriodicTickClearedOnRemoval covers FR-6.1/FR-6.2: every removal path
// drops the (character, statType) throttle entry. The predecessor poison
// throttle's zero-caller state must not recur.
func TestPeriodicTickClearedOnRemoval(t *testing.T) {
	const characterId = uint32(100)
	const sourceId = int32(1311008)
	key := TickKey{CharacterId: characterId, StatType: "DRAGON_BLOOD"}

	tests := []struct {
		name   string
		remove func(t *testing.T, p *ProcessorImpl)
	}{
		{"cancel", func(t *testing.T, p *ProcessorImpl) {
			require.NoError(t, p.Cancel(world.Id(0), characterId, sourceId))
		}},
		{"cancel all", func(t *testing.T, p *ProcessorImpl) {
			require.NoError(t, p.CancelAll(world.Id(0), characterId))
		}},
		{"cancel by stat types", func(t *testing.T, p *ProcessorImpl) {
			require.NoError(t, p.CancelByStatTypes(world.Id(0), characterId, []string{"DRAGON_BLOOD"}))
		}},
		{"expire for character", func(t *testing.T, p *ProcessorImpl) {
			require.NoError(t, p.ExpireForCharacter(world.Id(0), characterId))
		}},
		{"expire buffs sweep", func(t *testing.T, p *ProcessorImpl) {
			require.NoError(t, p.ExpireBuffs())
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setupTestRegistry(t)
			emitted.Reset()
			ctx := setupTestContext(t, setupTestTenant(t))
			now := time.Now()
			calls := 0
			p := tickProcessor(ctx, &now, 500, nil, &calls)

			// The expiry cases need a buff that has already lapsed; the cancel
			// cases need a live one. Duration is MILLISECONDS and must be > 0,
			// so "lapsed" is 1ms plus a sleep — Expired() reads the real clock.
			expiring := tc.name == "expire for character" || tc.name == "expire buffs sweep"
			duration := int32(600000)
			if expiring {
				duration = 1
			}
			_, err := GetRegistry().Apply(ctx, world.Id(0), channel.Id(1), characterId, sourceId, 1, duration,
				[]stat.Model{stat.NewStat("DRAGON_BLOOD", 48)}, false, false)
			require.NoError(t, err)
			if expiring {
				time.Sleep(10 * time.Millisecond)
			}

			GetRegistry().UpdatePeriodicTick(ctx, key, now)
			_, ok := GetRegistry().GetPeriodicTick(ctx, key)
			require.True(t, ok, "precondition: throttle entry exists")

			tc.remove(t, p)

			_, ok = GetRegistry().GetPeriodicTick(ctx, key)
			assert.False(t, ok, "removal path must clear the throttle entry")
		})
	}
}

// TestPeriodicTickRestartsAfterRecast: with the throttle cleared on cancel, a
// re-cast ticks immediately instead of waiting out the old schedule.
func TestPeriodicTickRestartsAfterRecast(t *testing.T) {
	setupTestRegistry(t)
	emitted.Reset()
	ctx := setupTestContext(t, setupTestTenant(t))
	now := time.Now()
	calls := 0
	p := tickProcessor(ctx, &now, 500, nil, &calls)

	applyBuff(t, ctx, 100, 1311008, stat.NewStat("DRAGON_BLOOD", 48))
	require.NoError(t, p.ProcessPeriodicTicks())
	require.Equal(t, []int16{-48}, changeHPAmounts(t))

	require.NoError(t, p.Cancel(world.Id(0), 100, 1311008))
	emitted.Reset()

	now = now.Add(time.Second) // well inside the 4s interval
	applyBuff(t, ctx, 100, 1311008, stat.NewStat("DRAGON_BLOOD", 48))
	require.NoError(t, p.ProcessPeriodicTicks())
	assert.Equal(t, []int16{-48}, changeHPAmounts(t), "cleared throttle means the re-cast ticks at once")
}
