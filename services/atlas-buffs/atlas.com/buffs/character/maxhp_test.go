package character

import (
	"testing"
	"time"

	"atlas-buffs/berserk"
	"atlas-buffs/buff/stat"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	constants "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func setupBothRegistries(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(client)
	berserk.InitRegistry(client)
}

func TestAffectsMaxHp(t *testing.T) {
	cases := []struct {
		name    string
		changes []stat.Model
		want    bool
	}{
		{name: "hyper body hp", changes: []stat.Model{stat.NewStat(string(constants.TemporaryStatTypeHyperBodyHP), 60)}, want: true},
		{name: "hyper body mp only", changes: []stat.Model{stat.NewStat(string(constants.TemporaryStatTypeHyperBodyMP), 60)}, want: false},
		{name: "plain stat buff", changes: []stat.Model{stat.NewStat("STR", 10)}, want: false},
		{name: "mixed includes max hp", changes: []stat.Model{stat.NewStat("STR", 10), stat.NewStat(string(constants.TemporaryStatTypeHyperBodyHP), 60)}, want: true},
		{name: "empty", changes: nil, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, affectsMaxHp(tc.changes))
		})
	}
}

func TestApplyHyperBodyMarksTrackedBerserkDirty(t *testing.T) {
	setupBothRegistries(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := logrus.New()

	assert.NoError(t, berserk.GetRegistry().Track(ctx,
		berserk.NewBuilder(world.Id(0), 42, 10).SetChannel(channel.Id(1)).Build()))

	changes := []stat.Model{stat.NewStat(string(constants.TemporaryStatTypeHyperBodyHP), 60)}
	assert.NoError(t, NewProcessor(l, ctx).Apply(world.Id(0), channel.Id(1), 42, 42, 1301007, 30, 10, changes, false))

	m, err := berserk.GetRegistry().Get(ctx, 42)
	assert.NoError(t, err)
	assert.False(t, m.DirtyAt().IsZero(), "hyper body apply marks berserk dirty")
	assert.False(t, m.DirtyDue(time.Now()), "grace-deferred: effective-stats must recompute first")
	assert.True(t, m.DirtyDue(time.Now().Add(berserk.ReevalGrace+time.Second)))
}

func TestCancelHyperBodyMarksTrackedBerserkDirty(t *testing.T) {
	setupBothRegistries(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := logrus.New()

	assert.NoError(t, berserk.GetRegistry().Track(ctx,
		berserk.NewBuilder(world.Id(0), 42, 10).SetChannel(channel.Id(1)).Build()))

	changes := []stat.Model{stat.NewStat(string(constants.TemporaryStatTypeHyperBodyHP), 60)}
	p := NewProcessor(l, ctx)
	assert.NoError(t, p.Apply(world.Id(0), channel.Id(1), 42, 42, 1301007, 30, 10, changes, false))

	// Clear the apply-time dirty mark so the cancel effect is observable.
	assert.NoError(t, berserk.GetRegistry().StoreEvaluation(ctx, 42, false, 100, time.Now()))
	_, claimed := berserk.GetRegistry().ClaimReeval(ctx, 42, time.Now().Add(berserk.ReevalGrace+time.Second))
	assert.True(t, claimed)

	assert.NoError(t, p.Cancel(world.Id(0), 42, 1301007))

	m, err := berserk.GetRegistry().Get(ctx, 42)
	assert.NoError(t, err)
	assert.False(t, m.DirtyAt().IsZero(), "hyper body cancel marks berserk dirty")
}

func TestApplyNonMaxHpBuffDoesNotMarkDirty(t *testing.T) {
	setupBothRegistries(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := logrus.New()

	assert.NoError(t, berserk.GetRegistry().Track(ctx,
		berserk.NewBuilder(world.Id(0), 42, 10).SetChannel(channel.Id(1)).Build()))

	changes := []stat.Model{stat.NewStat("STR", 10)}
	assert.NoError(t, NewProcessor(l, ctx).Apply(world.Id(0), channel.Id(1), 42, 42, 2001001, 30, 10, changes, false))

	m, err := berserk.GetRegistry().Get(ctx, 42)
	assert.NoError(t, err)
	assert.True(t, m.DirtyAt().IsZero())
}
