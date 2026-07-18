package berserk

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

func TestModelJSONRoundTrip(t *testing.T) {
	dirty := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	next := dirty.Add(5 * time.Second)
	m := NewBuilder(world.Id(1), 42, 10).
		SetChannel(channel.Id(2)).
		SetCharacterLevel(120).
		SetDirtyAt(dirty).
		Build()
	m = m.evaluated(true, 121, next)

	data, err := json.Marshal(m)
	assert.NoError(t, err)

	var got Model
	assert.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, world.Id(1), got.WorldId())
	assert.Equal(t, channel.Id(2), got.ChannelId())
	assert.True(t, got.ChannelKnown())
	assert.Equal(t, uint32(42), got.CharacterId())
	assert.Equal(t, byte(121), got.CharacterLevel())
	assert.Equal(t, byte(10), got.SkillLevel())
	assert.True(t, got.Active())
	assert.True(t, got.DirtyAt().Equal(dirty))
	assert.True(t, got.NextBroadcastAt().Equal(next))
}

func TestBuilderDefaults(t *testing.T) {
	m := NewBuilder(world.Id(0), 7, 1).Build()
	assert.False(t, m.ChannelKnown(), "channel unknown until a channel-bearing event")
	assert.False(t, m.Active())
	assert.True(t, m.DirtyAt().IsZero())
	assert.True(t, m.NextBroadcastAt().IsZero())
}

func TestMutators(t *testing.T) {
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	m := NewBuilder(world.Id(0), 7, 1).Build()

	m2 := m.channelUpdated(world.Id(1), channel.Id(3))
	assert.True(t, m2.ChannelKnown())
	assert.Equal(t, channel.Id(3), m2.ChannelId())
	assert.False(t, m.ChannelKnown(), "original unchanged (immutability)")

	m3 := m2.dirtyMarked(now)
	assert.True(t, m3.DirtyAt().Equal(now))
	m4 := m3.dirtyCleared()
	assert.True(t, m4.DirtyAt().IsZero())

	m5 := m4.skillLevelUpdated(20)
	assert.Equal(t, byte(20), m5.SkillLevel())

	m6 := m5.evaluated(true, 130, now.Add(5*time.Second))
	assert.True(t, m6.Active())
	assert.Equal(t, byte(130), m6.CharacterLevel())

	m7 := m6.broadcastScheduled(now.Add(3 * time.Second))
	assert.True(t, m7.NextBroadcastAt().Equal(now.Add(3*time.Second)))
}

func TestDueHelpers(t *testing.T) {
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	m := NewBuilder(world.Id(0), 7, 1).SetChannel(channel.Id(1)).Build()

	assert.False(t, m.DirtyDue(now), "zero dirtyAt = clean")
	assert.False(t, m.BroadcastDue(now), "zero nextBroadcastAt = not scheduled yet")

	assert.True(t, m.dirtyMarked(now).DirtyDue(now), "dirtyAt == now is due")
	assert.False(t, m.dirtyMarked(now.Add(time.Second)).DirtyDue(now), "future dirtyAt (grace) not due")

	sched := m.broadcastScheduled(now)
	assert.True(t, sched.BroadcastDue(now))

	unknown := NewBuilder(world.Id(0), 8, 1).Build().dirtyMarked(now).broadcastScheduled(now)
	assert.False(t, unknown.DirtyDue(now), "re-eval needs channelKnown (effective-stats route needs channel)")
	assert.False(t, unknown.BroadcastDue(now), "broadcast needs channelKnown (cannot route)")
}
