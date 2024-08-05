package monster

import (
	"atlas-maps/map/character"
	"atlas-maps/monster"
	"atlas-maps/tenant"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"math"
	"math/rand"
	"time"
)

func Spawn(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32) {
	return func(worldId byte, channelId byte, mapId uint32) {
		l.Debugf("Executing spawn mechanism for Tenant [%s] World [%d] Channel [%d] Map [%d].", tenant.String(), worldId, channelId, mapId)

		cs, err := character.GetCharactersInMap(l, span, tenant)(worldId, channelId, mapId)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve characters in map. Aborting spawning for world [%d] channel [%d] map [%d].", worldId, channelId, mapId)
			return
		}

		c := len(cs)
		if c > 0 {
			sps, err := GetSpawnPoints(l, span, tenant)(mapId)
			if err != nil {
				l.WithError(err).Errorf("Unable to get spawn points for map %d.", mapId)
				return
			}

			var ableSps []SpawnPoint
			for _, x := range sps {
				if x.MobTime >= 0 {
					ableSps = append(ableSps, x)
				}
			}

			monstersInMap, err := monster.CountInMap(l, span, tenant)(worldId, channelId, mapId)
			if err != nil {
				l.WithError(err).Warnf("Assuming no monsters in map.")
			}

			monstersMax := getMonsterMax(c, len(ableSps))

			toSpawn := monstersMax - monstersInMap
			if toSpawn > 0 {
				result := shuffle(ableSps)
				for i := 0; i < toSpawn; i++ {
					x := result[i]
					monster.CreateMonster(l, span, tenant)(worldId, channelId, mapId, x.Id, x.X, x.Y, x.Fh, x.Team)
				}
			}
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

func GetSpawnPoints(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(mapId uint32) ([]SpawnPoint, error) {
	return func(mapId uint32) ([]SpawnPoint, error) {
		return requests.SliceProvider[RestModel, SpawnPoint](l)(requestSpawnPoints(l, span, tenant)(mapId), Extract)()
	}
}
