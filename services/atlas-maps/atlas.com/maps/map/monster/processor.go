package monster

import (
	"atlas-maps/map/character"
	"atlas-maps/monster"
	"atlas-maps/tenant"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
	"math"
	"math/rand"
	"time"
)

func Spawn(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32) {
	return func(worldId byte, channelId byte, mapId uint32) {
		l.Debugf("Executing spawn mechanism for Tenant [%s] World [%d] Channel [%d] Map [%d].", tenant.String(), worldId, channelId, mapId)

		cs, err := character.GetCharactersInMap(tenant)(worldId, channelId, mapId)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve characters in map. Aborting spawning for world [%d] channel [%d] map [%d].", worldId, channelId, mapId)
			return
		}

		c := len(cs)
		if c < 0 {
			return
		}

		ableSps, err := SpawnableSpawnPointProvider(l, ctx, tenant)(mapId)()
		if err != nil {
			return
		}

		monstersInMap, err := monster.CountInMap(l, ctx, tenant)(worldId, channelId, mapId)
		if err != nil {
			l.WithError(err).Warnf("Assuming no monsters in map.")
		}

		monstersMax := getMonsterMax(c, len(ableSps))

		toSpawn := monstersMax - monstersInMap
		if toSpawn <= 0 {
			return
		}

		ableSps = shuffle(ableSps)
		for i := 0; i < toSpawn; i++ {
			sp := ableSps[i]
			go func() {
				monster.CreateMonster(l, ctx, tenant)(worldId, channelId, mapId, sp.Template, sp.X, sp.Y, sp.Fh, sp.Team)
			}()
		}
	}
}

func shuffle(vals []SpawnPoint) []SpawnPoint {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	ret := make([]SpawnPoint, len(vals))
	perm := r.Perm(len(vals))
	for i, randIndex := range perm {
		ret[i] = vals[randIndex]
	}
	return ret
}

func getMonsterMax(characterCount int, spawnPointCount int) int {
	spawnRate := 0.70 + (0.05 * math.Min(6, float64(characterCount)))
	return int(math.Ceil(spawnRate * float64(spawnPointCount)))
}

func SpawnableSpawnPointProvider(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(mapId uint32) model.Provider[[]SpawnPoint] {
	return func(mapId uint32) model.Provider[[]SpawnPoint] {
		return model.FilteredProvider(SpawnPointProvider(l, ctx, tenant)(mapId), Spawnable)
	}
}

func Spawnable(point SpawnPoint) bool {
	return point.MobTime >= 0
}

func SpawnPointProvider(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(mapId uint32) model.Provider[[]SpawnPoint] {
	return func(mapId uint32) model.Provider[[]SpawnPoint] {
		return requests.SliceProvider[RestModel, SpawnPoint](l)(requestSpawnPoints(ctx, tenant)(mapId), Extract)
	}
}
