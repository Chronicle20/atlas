package _map

import (
	"atlas-messages/character"
	"atlas-messages/kafka/producer"
	"atlas-messages/map/data"
	"atlas-messages/portal"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func Exists(l logrus.FieldLogger) func(ctx context.Context) func(mapId uint32) bool {
	return func(ctx context.Context) func(mapId uint32) bool {
		return func(mapId uint32) bool {
			_, err := data.GetById(l)(ctx)(mapId)
			if err != nil {
				l.WithError(err).Errorf("Unable to find requested map [%d].", mapId)
				return false
			}
			return true
		}
	}
}

func CharacterIdsInMapModelProvider(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32) model.Provider[[]uint32] {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32) model.Provider[[]uint32] {
		return func(worldId byte, channelId byte, mapId uint32) model.Provider[[]uint32] {
			return requests.SliceProvider[RestModel, uint32](l, ctx)(requestCharactersInMap(worldId, channelId, mapId), Extract, model.Filters[uint32]())
		}
	}
}

func GetCharacterIdsInMap(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32) ([]uint32, error) {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32) ([]uint32, error) {
		return func(worldId byte, channelId byte, mapId uint32) ([]uint32, error) {
			return CharacterIdsInMapModelProvider(l)(ctx)(worldId, channelId, mapId)()
		}
	}
}

func WarpRandom(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte) func(channelId byte) func(characterId uint32) func(mapId uint32) error {
	return func(ctx context.Context) func(worldId byte) func(channelId byte) func(characterId uint32) func(mapId uint32) error {
		return func(worldId byte) func(channelId byte) func(characterId uint32) func(mapId uint32) error {
			return func(channelId byte) func(characterId uint32) func(mapId uint32) error {
				return func(characterId uint32) func(mapId uint32) error {
					return func(mapId uint32) error {
						return WarpToPortal(l)(ctx)(worldId, channelId, characterId, mapId, portal.RandomSpawnPointIdProvider(l)(ctx)(mapId))
					}
				}
			}
		}
	}
}

func WarpToPortal(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32, mapId uint32, p model.Provider[uint32]) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32, mapId uint32, p model.Provider[uint32]) error {
		return func(worldId byte, channelId byte, characterId uint32, mapId uint32, p model.Provider[uint32]) error {
			id, err := p()
			if err != nil {
				return err
			}

			return producer.ProviderImpl(l)(ctx)(character.EnvCommandTopic)(character.ChangeMapProvider(worldId, channelId, characterId, mapId, id))
		}
	}
}
