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
	assert.True(t, m.NextBroadcastAt().Equal(now.Add(InitialBroadcastDelay)), "fresh 5s schedule")
}

// TestProcessTicksReevalDoesNotBroadcastSamePass pins that a SUCCESSFUL
// re-evaluation does not also broadcast in the same pass. The load-bearing
// assertion is about emission, which producertest.InstallNoop makes a no-op,
// so this pins the invariant via observable schedule state instead: after
// the pass, NextBroadcastAt must be exactly now+InitialBroadcastDelay (the
// re-eval's fresh schedule, StoreEvaluation/registry.go), and NOT
// now+BroadcastPeriod (what a ClaimBroadcast claim would have left,
// registry.go ClaimBroadcast). ProcessTicks itself does not enforce
// "one or the other" via control flow (both `if e.DirtyDue` and
// `if e.BroadcastDue` run unconditionally against the pre-pass snapshot) —
// it is ClaimBroadcast's *fresh* re-read inside the Redis Update transaction
// that declines the broadcast, because by the time it runs
// nextBroadcastAt is already now+InitialBroadcastDelay (not due). Do not
// reintroduce a `continue` after the re-eval branch to "enforce" this
// mutual exclusion — it is already guaranteed by ClaimBroadcast's freshness
// check, and a `continue` there instead starves broadcasts during a
// sustained re-evaluation lookup failure (see
// TestProcessTicksLookupFailureStillBroadcasts).
func TestProcessTicksReevalDoesNotBroadcastSamePass(t *testing.T) {
	setupTestRegistry(t)
	ctx := setupTestContext(t, setupTestTenant(t))
	now := time.Now()
	p := testProcessor(t, ctx, now)

	// Dirty AND broadcast-due: the re-evaluation succeeds and replaces
	// the schedule (design D2 cancel-and-replace semantics); the broadcast
	// claim must decline because the fresh schedule is no longer due.
	m := NewBuilder(world.Id(0), 42, 10).SetChannel(channel.Id(1)).SetDirtyAt(now).Build().
		evaluated(false, 120, now)
	assert.NoError(t, GetRegistry().Track(ctx, m))

	assert.NoError(t, p.ProcessTicks())

	got, _ := GetRegistry().Get(ctx, 42)
	assert.True(t, got.NextBroadcastAt().Equal(now.Add(InitialBroadcastDelay)),
		"schedule is the re-evaluation's fresh now+InitialBroadcastDelay")
	assert.False(t, got.NextBroadcastAt().Equal(now.Add(BroadcastPeriod)),
		"schedule must NOT be now+BroadcastPeriod, which is what a broadcast claim advancing the "+
			"pre-reeval schedule would have produced")
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
