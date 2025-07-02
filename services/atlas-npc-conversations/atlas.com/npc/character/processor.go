package character

import (
	"atlas-npc-conversations/job"
	character2 "atlas-npc-conversations/kafka/message/character"
	"atlas-npc-conversations/kafka/producer"
	"atlas-npc-conversations/portal"
	"context"
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

func RequestChangeMeso(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, worldId byte, amount int32) error {
	return func(ctx context.Context) func(characterId uint32, worldId byte, amount int32) error {
		return func(characterId uint32, worldId byte, amount int32) error {
			l.Debugf("Requesting to change character [%d] meso by [%d].", characterId, amount)
			return producer.ProviderImpl(l)(ctx)(character2.EnvCommandTopic)(requestChangeMesoCommandProvider(characterId, worldId, amount))
		}
	}
}

func WarpToPortal(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32, mapId uint32, p model.Provider[uint32]) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32, mapId uint32, p model.Provider[uint32]) error {
		return func(worldId byte, channelId byte, characterId uint32, mapId uint32, p model.Provider[uint32]) error {
			pid, err := p()
			if err != nil {
				return err
			}

			return producer.ProviderImpl(l)(ctx)(character2.EnvCommandTopic)(changeMapProvider(worldId, channelId, characterId, mapId, pid))
		}
	}
}

func WarpRandom(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32, mapId uint32) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32, mapId uint32) error {
		return func(worldId byte, channelId byte, characterId uint32, mapId uint32) error {
			return WarpToPortal(l)(ctx)(worldId, channelId, characterId, mapId, portal.RandomPortalIdProvider(l)(ctx)(mapId))
		}
	}
}

func WarpById(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32, mapId uint32, portalId uint32) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32, mapId uint32, portalId uint32) error {
		return func(worldId byte, channelId byte, characterId uint32, mapId uint32, portalId uint32) error {
			return WarpToPortal(l)(ctx)(worldId, channelId, characterId, mapId, model.FixedProvider(portalId))
		}
	}
}

func WarpByName(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32, mapId uint32, portalName string) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32, mapId uint32, portalName string) error {
		return func(worldId byte, channelId byte, characterId uint32, mapId uint32, portalName string) error {
			return WarpToPortal(l)(ctx)(worldId, channelId, characterId, mapId, portal.ByNamePortalIdProvider(l)(ctx)(mapId, portalName))
		}
	}
}
