package charstatus

import (
	"atlas-buffs/buff/stat"
	"atlas-buffs/character"
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func setupTest(t *testing.T) context.Context {
	t.Helper()
	mr := miniredis.RunT(t)
	character.InitRegistry(goredis.NewClient(&goredis.Options{Addr: mr.Addr()}))
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	assert.NoError(t, err)
	return tenant.WithContext(context.Background(), ten)
}

func seedBeacon(t *testing.T, ctx context.Context, characterId uint32) {
	t.Helper()
	_, err := character.GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), characterId, int32(5211006), byte(1),
		0, []stat.Model{stat.NewStat(string(charconst.TemporaryStatTypeHomingBeacon), 1000001)}, false, true)
	assert.NoError(t, err)
}

func TestHandleMapChanged_CancelsBeacon(t *testing.T) {
	ctx := setupTest(t)
	characterId := uint32(2000)
	seedBeacon(t, ctx, characterId)

	handleMapChanged(logrus.StandardLogger(), ctx, StatusEvent[MapChangedBody]{
		WorldId: world.Id(0), CharacterId: characterId, Type: StatusEventTypeMapChanged,
	})

	m, err := character.GetRegistry().Get(ctx, characterId)
	assert.NoError(t, err)
	assert.Len(t, m.Buffs(), 0)
}

// A non-MAP_CHANGED event on the same topic must be ignored.
func TestHandleMapChanged_GuardsType(t *testing.T) {
	ctx := setupTest(t)
	characterId := uint32(2001)
	seedBeacon(t, ctx, characterId)

	handleMapChanged(logrus.StandardLogger(), ctx, StatusEvent[MapChangedBody]{
		WorldId: world.Id(0), CharacterId: characterId, Type: "LOGOUT",
	})

	m, err := character.GetRegistry().Get(ctx, characterId)
	assert.NoError(t, err)
	assert.Len(t, m.Buffs(), 1)
}

// Only HOMING_BEACON is canceled on map change — other buffs survive.
func TestHandleMapChanged_LeavesOtherBuffs(t *testing.T) {
	ctx := setupTest(t)
	characterId := uint32(2002)
	seedBeacon(t, ctx, characterId)
	_, err := character.GetRegistry().Apply(ctx, world.Id(0), channel.Id(0), characterId, int32(2001001), byte(5),
		60000, []stat.Model{stat.NewStat("SPEED", 20)}, false, false)
	assert.NoError(t, err)

	handleMapChanged(logrus.StandardLogger(), ctx, StatusEvent[MapChangedBody]{
		WorldId: world.Id(0), CharacterId: characterId, Type: StatusEventTypeMapChanged,
	})

	m, err := character.GetRegistry().Get(ctx, characterId)
	assert.NoError(t, err)
	assert.Len(t, m.Buffs(), 1)
	for _, b := range m.Buffs() {
		assert.Equal(t, int32(2001001), b.SourceId())
	}
}
