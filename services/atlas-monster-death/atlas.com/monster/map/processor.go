package _map

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

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
