package character

import (
	"atlas-npc-conversations/job"
	character2 "atlas-npc-conversations/kafka/message/character"
	"atlas-npc-conversations/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func GetById(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) (Model, error) {
	return func(ctx context.Context) func(characterId uint32) (Model, error) {
		return func(characterId uint32) (Model, error) {
			return requests.Provider[RestModel, Model](l, ctx)(requestById(characterId), Extract)()
		}
	}
}

type AttributeCriteria func(Model) bool

func MeetsCriteria(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, criteria ...AttributeCriteria) bool {
	return func(ctx context.Context) func(characterId uint32, criteria ...AttributeCriteria) bool {
		return func(characterId uint32, criteria ...AttributeCriteria) bool {
			c, err := GetById(l)(ctx)(characterId)
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve character %d for criteria check.", characterId)
				return false
			}
			for _, check := range criteria {
				if ok := check(c); !ok {
					return false
				}
			}
			return true
		}
	}
}

func IsBeginnerTree(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) bool {
	return func(ctx context.Context) func(characterId uint32) bool {
		return func(characterId uint32) bool {
			return MeetsCriteria(l)(ctx)(characterId, IsBeginnerTreeCriteria())
		}
	}
}

func IsBeginnerTreeCriteria() AttributeCriteria {
	return func(c Model) bool {
		return job.IsA(c.JobId(), job.Beginner, job.Noblesse, job.Legend)
	}
}

func HasMeso(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, amount uint32) bool {
	return func(ctx context.Context) func(characterId uint32, amount uint32) bool {
		return func(characterId uint32, amount uint32) bool {
			return MeetsCriteria(l)(ctx)(characterId, HasMesoCriteria(amount))
		}
	}
}

func HasMesoCriteria(amount uint32) AttributeCriteria {
	return func(c Model) bool {
		return c.Meso() >= amount
	}
}

func RequestChangeMeso(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, worldId world.Id, amount int32) error {
	return func(ctx context.Context) func(characterId uint32, worldId world.Id, amount int32) error {
		return func(characterId uint32, worldId world.Id, amount int32) error {
			l.Debugf("Requesting to change character [%d] meso by [%d].", characterId, amount)
			return producer.ProviderImpl(l)(ctx)(character2.EnvCommandTopic)(requestChangeMesoCommandProvider(characterId, byte(worldId), amount))
		}
	}
}

func WarpToPortal(l logrus.FieldLogger) func(ctx context.Context) func(worldId world.Id, channelId channel.Id, characterId uint32, mapId uint32, p model.Provider[uint32]) error {
	return func(ctx context.Context) func(worldId world.Id, channelId channel.Id, characterId uint32, mapId uint32, p model.Provider[uint32]) error {
		return func(worldId world.Id, channelId channel.Id, characterId uint32, mapId uint32, p model.Provider[uint32]) error {
			pid, err := p()
			if err != nil {
				return err
			}

			return producer.ProviderImpl(l)(ctx)(character2.EnvCommandTopic)(changeMapProvider(byte(worldId), byte(channelId), characterId, mapId, pid))
		}
	}
}

func WarpById(l logrus.FieldLogger) func(ctx context.Context) func(worldId world.Id, channelId channel.Id, characterId uint32, mapId uint32, portalId uint32) error {
	return func(ctx context.Context) func(worldId world.Id, channelId channel.Id, characterId uint32, mapId uint32, portalId uint32) error {
		return func(worldId world.Id, channelId channel.Id, characterId uint32, mapId uint32, portalId uint32) error {
			return WarpToPortal(l)(ctx)(worldId, channelId, characterId, mapId, model.FixedProvider(portalId))
		}
	}
}
