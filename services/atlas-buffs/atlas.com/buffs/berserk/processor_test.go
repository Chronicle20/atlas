package berserk

import (
	"context"
	"errors"
	"testing"
	"time"

	extchar "atlas-buffs/external/character"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/stat"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// testProcessor builds a ProcessorImpl with deterministic time and stubbed
// externals. Same-package construction — no test helpers file (project rule);
// the Builder pattern is used for all Model setup.
func testProcessor(t *testing.T, ctx context.Context, now time.Time) *ProcessorImpl {
	t.Helper()
	return &ProcessorImpl{
		l:   logrus.New(),
		ctx: ctx,
		now: func() time.Time { return now },
		getCharacter: func(characterId uint32) (extchar.RestModel, error) {
			return extchar.RestModel{Id: characterId, Level: 120, Hp: 100}, nil
		},
		getSkillLevel: func(characterId uint32) (byte, error) { return 10, nil },
		getMaxHp:      func(worldId world.Id, channelId channel.Id, characterId uint32) (uint32, error) { return 1000, nil },
		getEffectX:    func(skillLevel byte) (int16, error) { return 30, nil },
	}
}

func TestTrackOnLoginSkillLevelZeroNotTracked(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))
	p := testProcessor(t, ctx, time.Now())
	p.getSkillLevel = func(uint32) (byte, error) { return 0, nil }

	assert.NoError(t, p.TrackOnLogin(world.Id(0), channel.Id(1), 42))
	_, err := GetRegistry().Get(ctx, 42)
	assert.ErrorIs(t, err, ErrNotFound, "level 0 characters generate no registry entries")
}

func TestTrackOnLoginTracksAndMarksDirty(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))
	now := time.Now()
	p := testProcessor(t, ctx, now)

	assert.NoError(t, p.TrackOnLogin(world.Id(0), channel.Id(1), 42))
	m, err := GetRegistry().Get(ctx, 42)
	assert.NoError(t, err)
	assert.Equal(t, byte(10), m.SkillLevel())
	assert.True(t, m.ChannelKnown())
	assert.True(t, m.DirtyDue(now))
}

func TestHandleStatChanged(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))
	now := time.Now()
	p := testProcessor(t, ctx, now)

	// Untracked: zero work, no error.
	assert.NoError(t, p.HandleStatChanged(world.Id(0), channel.Id(1), 99, []stat.Type{stat.TypeHp}))

	assert.NoError(t, GetRegistry().Track(ctx, NewBuilder(world.Id(0), 42, 10).SetChannel(channel.Id(1)).Build()))

	// Non-HP updates refresh channel but do not mark dirty.
	assert.NoError(t, p.HandleStatChanged(world.Id(0), channel.Id(2), 42, []stat.Type{stat.TypeStrength}))
	m, _ := GetRegistry().Get(ctx, 42)
	assert.Equal(t, channel.Id(2), m.ChannelId())
	assert.True(t, m.DirtyAt().IsZero())

	// HP update: dirty now.
	assert.NoError(t, p.HandleStatChanged(world.Id(0), channel.Id(2), 42, []stat.Type{stat.TypeHp}))
	m, _ = GetRegistry().Get(ctx, 42)
	assert.True(t, m.DirtyAt().Equal(now))

	// MAX_HP present: grace-deferred even when HP is also present (the
	// max-HP recompute in effective-stats is what we are waiting out).
	assert.NoError(t, p.HandleStatChanged(world.Id(0), channel.Id(2), 42, []stat.Type{stat.TypeHp, stat.TypeMaxHp}))
	m, _ = GetRegistry().Get(ctx, 42)
	assert.True(t, m.DirtyAt().Equal(now.Add(ReevalGrace)))
}

func TestHandleSkillUpdated(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))
	now := time.Now()
	p := testProcessor(t, ctx, now)

	// New (SP allocation 0→1): tracked without channel.
	assert.NoError(t, p.HandleSkillUpdated(world.Id(0), 42, 1))
	m, err := GetRegistry().Get(ctx, 42)
	assert.NoError(t, err)
	assert.Equal(t, byte(1), m.SkillLevel())
	assert.False(t, m.ChannelKnown())
	assert.True(t, m.DirtyAt().Equal(now))

	// Existing: level refresh + dirty.
	assert.NoError(t, GetRegistry().UpdateChannel(ctx, 42, world.Id(0), channel.Id(1)))
	assert.NoError(t, p.HandleSkillUpdated(world.Id(0), 42, 2))
	m, _ = GetRegistry().Get(ctx, 42)
	assert.Equal(t, byte(2), m.SkillLevel())
	assert.True(t, m.ChannelKnown(), "level update must not lose the channel")

	// Level 0 (SP reset): untracked.
	assert.NoError(t, p.HandleSkillUpdated(world.Id(0), 42, 0))
	_, err = GetRegistry().Get(ctx, 42)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestProcessTicksReevaluates(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))
	now := time.Now()
	p := testProcessor(t, ctx, now)
	// hp=100, maxHp=1000, x=30 → 10 < 30 → active.

	assert.NoError(t, GetRegistry().Track(ctx,
		NewBuilder(world.Id(0), 42, 10).SetChannel(channel.Id(1)).SetDirtyAt(now).Build()))

	assert.NoError(t, p.ProcessTicks())

	m, err := GetRegistry().Get(ctx, 42)
	assert.NoError(t, err)
	assert.True(t, m.Active())
	assert.Equal(t, byte(120), m.CharacterLevel(), "character level refreshed from REST")
	assert.True(t, m.DirtyAt().IsZero(), "claim cleared")
	assert.True(t, m.NextBroadcastAt().Equal(now), "first evaluation broadcasts promptly (no initial delay)")
}

// TestProcessTicksReevalUnchangedStatePreservesSchedule is the core regression
// test for the aura-starvation fix (task-154 live-test finding). When a
// re-evaluation finds the berserk state UNCHANGED, it must leave the running
// broadcast schedule alone. Before the fix, every re-evaluation reset
// nextBroadcastAt to now+InitialBroadcastDelay, so a stream of HP STAT_CHANGED
// events (sustained combat) pushed the broadcast deadline out on every pass and
// the aura never broadcast until HP stopped changing.
func TestProcessTicksReevalUnchangedStatePreservesSchedule(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))
	now := time.Now()
	p := testProcessor(t, ctx, now)
	// testProcessor evaluates active=true (hp 100 / maxHp 1000 = 10% < x 30).

	// Already active, mid-cadence (a broadcast scheduled 2s out), and dirty (an
	// HP change just arrived). The re-eval recomputes active=true — unchanged —
	// so it must NOT touch the schedule.
	sched := now.Add(2 * time.Second)
	m := NewBuilder(world.Id(0), 42, 10).SetChannel(channel.Id(1)).SetDirtyAt(now).Build().
		evaluated(true, 120, sched)
	assert.NoError(t, GetRegistry().Track(ctx, m))

	assert.NoError(t, p.ProcessTicks())

	got, _ := GetRegistry().Get(ctx, 42)
	assert.True(t, got.Active(), "still active")
	assert.True(t, got.DirtyAt().IsZero(), "re-eval claim cleared the dirty flag")
	assert.True(t, got.NextBroadcastAt().Equal(sched),
		"unchanged state must NOT reset the broadcast schedule (anti-starvation)")
}

// TestProcessTicksReevalTransitionBroadcastsPromptly pins that a state change
// makes the aura flip promptly: the re-evaluation sets nextBroadcastAt to `now`
// so the next scan pass broadcasts the new state, instead of parking it behind a
// fresh delay. Here the last-known state is inactive with the schedule parked a
// minute out; the re-eval finds active=true.
func TestProcessTicksReevalTransitionBroadcastsPromptly(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))
	now := time.Now()
	p := testProcessor(t, ctx, now)
	// testProcessor evaluates active=true.

	future := now.Add(time.Minute)
	m := NewBuilder(world.Id(0), 42, 10).SetChannel(channel.Id(1)).SetDirtyAt(now).Build().
		evaluated(false, 120, future)
	assert.NoError(t, GetRegistry().Track(ctx, m))

	assert.NoError(t, p.ProcessTicks())

	got, _ := GetRegistry().Get(ctx, 42)
	assert.True(t, got.Active(), "flipped to active")
	assert.True(t, got.NextBroadcastAt().Equal(now),
		"a transition resets the schedule to now so the aura broadcasts on the next pass")
}

func TestProcessTicksBroadcastAdvancesSchedule(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))
	now := time.Now()
	p := testProcessor(t, ctx, now)

	m := NewBuilder(world.Id(0), 42, 10).SetChannel(channel.Id(1)).Build().
		evaluated(true, 120, now)
	assert.NoError(t, GetRegistry().Track(ctx, m))

	assert.NoError(t, p.ProcessTicks())

	got, _ := GetRegistry().Get(ctx, 42)
	assert.True(t, got.NextBroadcastAt().Equal(now.Add(BroadcastPeriod)))
	assert.True(t, got.Active(), "broadcast uses captured state, does not recompute")
}

func TestProcessTicksLookupFailureRearms(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))
	now := time.Now()
	p := testProcessor(t, ctx, now)
	p.getMaxHp = func(world.Id, channel.Id, uint32) (uint32, error) { return 0, errors.New("effective-stats down") }

	m := NewBuilder(world.Id(0), 42, 10).SetChannel(channel.Id(1)).SetDirtyAt(now).Build().
		evaluated(true, 120, now.Add(time.Minute))
	assert.NoError(t, GetRegistry().Track(ctx, m))

	assert.NoError(t, p.ProcessTicks(), "lookup failure never fails the pass")

	got, _ := GetRegistry().Get(ctx, 42)
	assert.True(t, got.DirtyAt().Equal(now.Add(ReevalRetryDelay)), "re-armed for retry")
	assert.True(t, got.Active(), "last-known state kept")
	assert.True(t, got.NextBroadcastAt().Equal(now.Add(time.Minute)), "existing schedule untouched")
}

// TestProcessTicksLookupFailureStillBroadcasts pins FR-5: an entry that is
// BOTH dirty-due AND broadcast-due at the same pass must still broadcast its
// last-known state when the re-evaluation lookup fails. Without this, a
// sustained REST outage (ReevalRetryDelay == the ticker's scan interval)
// re-arms dirtyAt to exactly the next pass forever, so the re-evaluation
// branch claims the entry on every single pass and the broadcast branch is
// never reached — the aura freezes for the whole outage. Unlike
// TestProcessTicksLookupFailureRearms (which parks nextBroadcastAt a minute
// out so it never comes due, and so cannot exercise this path), this test
// sets nextBroadcastAt to `now` so both branches are due on the same pass.
func TestProcessTicksLookupFailureStillBroadcasts(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))
	now := time.Now()
	p := testProcessor(t, ctx, now)
	p.getMaxHp = func(world.Id, channel.Id, uint32) (uint32, error) { return 0, errors.New("effective-stats down") }

	// Dirty-due AND broadcast-due at `now`, with a known last-evaluated Active
	// state.
	m := NewBuilder(world.Id(0), 42, 10).SetChannel(channel.Id(1)).SetDirtyAt(now).Build().
		evaluated(true, 120, now)
	assert.NoError(t, GetRegistry().Track(ctx, m))

	assert.NoError(t, p.ProcessTicks(), "lookup failure never fails the pass")

	got, _ := GetRegistry().Get(ctx, 42)
	assert.True(t, got.DirtyAt().Equal(now.Add(ReevalRetryDelay)), "re-armed for retry")
	assert.True(t, got.NextBroadcastAt().Equal(now.Add(BroadcastPeriod)),
		"broadcast still happened despite the failed re-eval, advancing the schedule")
	assert.True(t, got.Active(), "last-known state preserved and broadcast")
}

func TestProcessTicksMaxHpZeroGuard(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))
	now := time.Now()
	p := testProcessor(t, ctx, now)
	p.getMaxHp = func(world.Id, channel.Id, uint32) (uint32, error) { return 0, nil }

	assert.NoError(t, GetRegistry().Track(ctx,
		NewBuilder(world.Id(0), 42, 10).SetChannel(channel.Id(1)).SetDirtyAt(now).Build()))

	assert.NoError(t, p.ProcessTicks())
	got, _ := GetRegistry().Get(ctx, 42)
	assert.True(t, got.DirtyAt().Equal(now.Add(ReevalRetryDelay)), "maxHp=0 treated as failed lookup")
}
