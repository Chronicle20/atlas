package buff

import (
	buffmsg "atlas-mounts/kafka/message/buff"
	"atlas-mounts/mount"
	"context"
	"errors"
	"testing"

	characterconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

const monsterRiding = string(characterconst.TemporaryStatTypeMonsterRiding)

// fakeSeams swaps the registry/processor seams for in-memory recorders and
// restores them when the returned cleanup runs.
type fakeSeams struct {
	addCalls    []mount.MountRideContext
	addCharIds  []uint32
	removeCalls []uint32
	setCalls    int
	setWorlds   []world.Id
	setCharIds  []uint32
}

func newFake(t *testing.T) *fakeSeams {
	t.Helper()
	f := &fakeSeams{}

	origAdd := registryAdd
	origRemove := registryRemove
	origSet := emitSet

	registryAdd = func(_ context.Context, characterId uint32, c mount.MountRideContext) error {
		f.addCharIds = append(f.addCharIds, characterId)
		f.addCalls = append(f.addCalls, c)
		return nil
	}
	registryRemove = func(_ context.Context, characterId uint32) error {
		f.removeCalls = append(f.removeCalls, characterId)
		return nil
	}
	emitSet = func(_ logrus.FieldLogger, _ context.Context, _ *gorm.DB, worldId world.Id, characterId uint32) error {
		f.setCalls++
		f.setWorlds = append(f.setWorlds, worldId)
		f.setCharIds = append(f.setCharIds, characterId)
		return nil
	}

	t.Cleanup(func() {
		registryAdd = origAdd
		registryRemove = origRemove
		emitSet = origSet
	})
	return f
}

func testLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)
	return l
}

func TestHandleBuffApplied_TamedMount(t *testing.T) {
	f := newFake(t)

	e := buffmsg.StatusEvent[buffmsg.AppliedStatusEventBody]{
		WorldId:     world.Id(7),
		CharacterId: 1000,
		Type:        buffmsg.EventStatusTypeBuffApplied,
		Body: buffmsg.AppliedStatusEventBody{
			SourceId: 1004, // BeginnerMonsterRidingId (tamed)
			Changes: []buffmsg.StatChange{
				{Type: monsterRiding, Amount: 1902000},
			},
		},
	}

	handleBuffApplied(nil)(testLogger(), context.Background(), e)

	assert.Len(t, f.addCalls, 1, "registry.Add must be called once for a tamed mount")
	assert.Equal(t, uint32(1000), f.addCharIds[0])
	assert.Equal(t, world.Id(7), f.addCalls[0].WorldId)
	assert.Equal(t, int32(1004), f.addCalls[0].SkillId)
	assert.Equal(t, int32(1902000), f.addCalls[0].VehicleId)
	assert.Equal(t, 1, f.setCalls, "EmitSet must be invoked")
	assert.Equal(t, world.Id(7), f.setWorlds[0])
	assert.Equal(t, uint32(1000), f.setCharIds[0])
	assert.Empty(t, f.removeCalls)
}

func TestHandleBuffApplied_SkillOnlyMount(t *testing.T) {
	f := newFake(t)

	e := buffmsg.StatusEvent[buffmsg.AppliedStatusEventBody]{
		WorldId:     world.Id(1),
		CharacterId: 2000,
		Type:        buffmsg.EventStatusTypeBuffApplied,
		Body: buffmsg.AppliedStatusEventBody{
			SourceId: 1019, // BeginnerBroomstickId (skill-only)
			Changes: []buffmsg.StatChange{
				{Type: monsterRiding, Amount: 1932005},
			},
		},
	}

	handleBuffApplied(nil)(testLogger(), context.Background(), e)

	assert.Empty(t, f.addCalls, "skill-only mounts must NOT be added to the registry")
	assert.Equal(t, 1, f.setCalls, "EmitSet must still be invoked for skill-only mounts")
	assert.Equal(t, uint32(2000), f.setCharIds[0])
}

func TestHandleBuffApplied_NonMount(t *testing.T) {
	f := newFake(t)

	e := buffmsg.StatusEvent[buffmsg.AppliedStatusEventBody]{
		WorldId:     world.Id(1),
		CharacterId: 3000,
		Type:        buffmsg.EventStatusTypeBuffApplied,
		Body: buffmsg.AppliedStatusEventBody{
			SourceId: 1101006, // some attack buff
			Changes: []buffmsg.StatChange{
				{Type: "WEAPON_ATTACK", Amount: 10},
			},
		},
	}

	handleBuffApplied(nil)(testLogger(), context.Background(), e)

	assert.Empty(t, f.addCalls, "non-mount buff must not touch the registry")
	assert.Equal(t, 0, f.setCalls, "non-mount buff must not emit SET")
	assert.Empty(t, f.removeCalls)
}

func TestHandleBuffApplied_WrongType(t *testing.T) {
	f := newFake(t)

	e := buffmsg.StatusEvent[buffmsg.AppliedStatusEventBody]{
		WorldId:     world.Id(1),
		CharacterId: 3001,
		Type:        "SOMETHING_ELSE",
		Body: buffmsg.AppliedStatusEventBody{
			SourceId: 1004,
			Changes:  []buffmsg.StatChange{{Type: monsterRiding, Amount: 1902000}},
		},
	}

	handleBuffApplied(nil)(testLogger(), context.Background(), e)

	assert.Empty(t, f.addCalls)
	assert.Equal(t, 0, f.setCalls)
}

func TestHandleBuffExpired_Mount(t *testing.T) {
	f := newFake(t)

	e := buffmsg.StatusEvent[buffmsg.ExpiredStatusEventBody]{
		WorldId:     world.Id(1),
		CharacterId: 4000,
		Type:        buffmsg.EventStatusTypeBuffExpired,
		Body: buffmsg.ExpiredStatusEventBody{
			SourceId: 1004,
			Changes: []buffmsg.StatChange{
				{Type: monsterRiding, Amount: 1902000},
			},
		},
	}

	handleBuffExpired(nil)(testLogger(), context.Background(), e)

	assert.Equal(t, []uint32{4000}, f.removeCalls, "expired mount buff must remove the registry entry")
	assert.Empty(t, f.addCalls)
	assert.Equal(t, 0, f.setCalls)
}

func TestHandleBuffExpired_NonMount(t *testing.T) {
	f := newFake(t)

	e := buffmsg.StatusEvent[buffmsg.ExpiredStatusEventBody]{
		WorldId:     world.Id(1),
		CharacterId: 4001,
		Type:        buffmsg.EventStatusTypeBuffExpired,
		Body: buffmsg.ExpiredStatusEventBody{
			SourceId: 1101006,
			Changes:  []buffmsg.StatChange{{Type: "WEAPON_ATTACK", Amount: 10}},
		},
	}

	handleBuffExpired(nil)(testLogger(), context.Background(), e)

	assert.Empty(t, f.removeCalls, "non-mount expiry must not touch the registry")
}

func TestHandleBuffApplied_AddErrorSkipsSet(t *testing.T) {
	f := newFake(t)
	origAdd := registryAdd
	registryAdd = func(_ context.Context, _ uint32, _ mount.MountRideContext) error {
		return errors.New("boom")
	}
	t.Cleanup(func() { registryAdd = origAdd })

	e := buffmsg.StatusEvent[buffmsg.AppliedStatusEventBody]{
		WorldId:     world.Id(1),
		CharacterId: 5000,
		Type:        buffmsg.EventStatusTypeBuffApplied,
		Body: buffmsg.AppliedStatusEventBody{
			SourceId: 1004,
			Changes:  []buffmsg.StatChange{{Type: monsterRiding, Amount: 1902000}},
		},
	}

	handleBuffApplied(nil)(testLogger(), context.Background(), e)

	assert.Equal(t, 0, f.setCalls, "if registry.Add fails, SET must not be emitted")
}
