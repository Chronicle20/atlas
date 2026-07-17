package skillstatus

import (
	"context"
	"testing"

	"atlas-buffs/berserk"
	skillstatus2 "atlas-buffs/kafka/message/skillstatus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func setup(t *testing.T) context.Context {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	berserk.InitRegistry(client)
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	assert.NoError(t, err)
	return tenant.WithContext(context.Background(), ten)
}

func TestHandleUpdatedTracksBerserkSkill(t *testing.T) {
	ctx := setup(t)
	l := logrus.New()

	handleStatusEventUpdated(l, ctx, skillstatus2.StatusEvent[skillstatus2.StatusEventUpdatedBody]{
		WorldId: world.Id(0), CharacterId: 42, SkillId: uint32(skill.DarkKnightBerserkId),
		Type: skillstatus2.StatusEventTypeUpdated,
		Body: skillstatus2.StatusEventUpdatedBody{Level: 1},
	})

	m, err := berserk.GetRegistry().Get(ctx, 42)
	assert.NoError(t, err)
	assert.Equal(t, byte(1), m.SkillLevel())
	assert.False(t, m.ChannelKnown(), "skill events carry no channel (design D8)")
}

func TestHandleUpdatedIgnoresOtherSkills(t *testing.T) {
	ctx := setup(t)
	l := logrus.New()

	handleStatusEventUpdated(l, ctx, skillstatus2.StatusEvent[skillstatus2.StatusEventUpdatedBody]{
		WorldId: world.Id(0), CharacterId: 42, SkillId: uint32(skill.DarkKnightAchillesId),
		Type: skillstatus2.StatusEventTypeUpdated,
		Body: skillstatus2.StatusEventUpdatedBody{Level: 5},
	})

	assert.Empty(t, berserk.GetRegistry().GetAll(ctx))
}

func TestHandleUpdatedLevelZeroUntracks(t *testing.T) {
	ctx := setup(t)
	l := logrus.New()
	assert.NoError(t, berserk.GetRegistry().Track(ctx,
		berserk.NewBuilder(world.Id(0), 42, 10).SetChannel(channel.Id(1)).Build()))

	handleStatusEventUpdated(l, ctx, skillstatus2.StatusEvent[skillstatus2.StatusEventUpdatedBody]{
		WorldId: world.Id(0), CharacterId: 42, SkillId: uint32(skill.DarkKnightBerserkId),
		Type: skillstatus2.StatusEventTypeUpdated,
		Body: skillstatus2.StatusEventUpdatedBody{Level: 0},
	})

	_, err := berserk.GetRegistry().Get(ctx, 42)
	assert.ErrorIs(t, err, berserk.ErrNotFound)
}

func TestHandleDeletedUntracks(t *testing.T) {
	ctx := setup(t)
	l := logrus.New()
	assert.NoError(t, berserk.GetRegistry().Track(ctx,
		berserk.NewBuilder(world.Id(0), 42, 10).SetChannel(channel.Id(1)).Build()))

	handleStatusEventDeleted(l, ctx, skillstatus2.StatusEvent[skillstatus2.StatusEventDeletedBody]{
		WorldId: world.Id(0), CharacterId: 42, SkillId: uint32(skill.DarkKnightBerserkId),
		Type: skillstatus2.StatusEventTypeDeleted,
	})

	_, err := berserk.GetRegistry().Get(ctx, 42)
	assert.ErrorIs(t, err, berserk.ErrNotFound)
}
