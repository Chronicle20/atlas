package food

import (
	foodmsg "atlas-mounts/kafka/message/food"
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// fakeFeed swaps the applyFeed seam for an in-memory recorder and restores it
// when the returned cleanup runs.
type fakeFeed struct {
	calls     int
	worldIds  []world.Id
	charIds   []uint32
	healMaxes []int
}

func newFakeFeed(t *testing.T) *fakeFeed {
	t.Helper()
	f := &fakeFeed{}

	orig := applyFeed
	applyFeed = func(_ logrus.FieldLogger, _ context.Context, _ *gorm.DB, worldId world.Id, characterId uint32, healMax int) error {
		f.calls++
		f.worldIds = append(f.worldIds, worldId)
		f.charIds = append(f.charIds, characterId)
		f.healMaxes = append(f.healMaxes, healMax)
		return nil
	}
	t.Cleanup(func() { applyFeed = orig })
	return f
}

func testLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)
	return l
}

func TestHandleTamingMobFood_RoutesToApplyFeed(t *testing.T) {
	f := newFakeFeed(t)

	e := foodmsg.Event{
		WorldId:       world.Id(7),
		CharacterId:   1000,
		ItemId:        2260000,
		TirednessHeal: 30,
	}

	handleTamingMobFood(nil)(testLogger(), context.Background(), e)

	assert.Equal(t, 1, f.calls, "ApplyFeedAndEmit must be invoked once")
	assert.Equal(t, world.Id(7), f.worldIds[0], "worldId must thread through")
	assert.Equal(t, uint32(1000), f.charIds[0], "characterId must thread through")
	assert.Equal(t, 30, f.healMaxes[0], "tirednessHeal must thread through as healMax")
}

func TestHandleTamingMobFood_DifferentHealThreadsThrough(t *testing.T) {
	f := newFakeFeed(t)

	e := foodmsg.Event{
		WorldId:       world.Id(2),
		CharacterId:   5555,
		ItemId:        2260001,
		TirednessHeal: 100,
	}

	handleTamingMobFood(nil)(testLogger(), context.Background(), e)

	assert.Equal(t, 1, f.calls)
	assert.Equal(t, world.Id(2), f.worldIds[0])
	assert.Equal(t, uint32(5555), f.charIds[0])
	assert.Equal(t, 100, f.healMaxes[0], "a different tirednessHeal must map to healMax")
}
