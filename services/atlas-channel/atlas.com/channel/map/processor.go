package _map

import (
	"atlas-channel/session"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/Chronicle20/atlas-tenant"
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

func ForSessionsInMap(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, o model.Operator[session.Model]) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, o model.Operator[session.Model]) error {
		return func(worldId byte, channelId byte, mapId uint32, o model.Operator[session.Model]) error {
			return session.ForEachByCharacterId(tenant.MustFromContext(ctx))(CharacterIdsInMapModelProvider(l)(ctx)(worldId, channelId, mapId), o)
		}
	}
}

func NotCharacterIdFilter(referenceCharacterId uint32) func(characterId uint32) bool {
	return func(characterId uint32) bool {
		return referenceCharacterId != characterId
	}
}

func OtherCharacterIdsInMapModelProvider(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, referenceCharacterId uint32) model.Provider[[]uint32] {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, referenceCharacterId uint32) model.Provider[[]uint32] {
		return func(worldId byte, channelId byte, mapId uint32, referenceCharacterId uint32) model.Provider[[]uint32] {
			return model.FilteredProvider(CharacterIdsInMapModelProvider(l)(ctx)(worldId, channelId, mapId), model.Filters(NotCharacterIdFilter(referenceCharacterId)))
		}
	}
}

func ForOtherSessionsInMap(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, referenceCharacterId uint32, o model.Operator[session.Model]) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, referenceCharacterId uint32, o model.Operator[session.Model]) error {
		return func(worldId byte, channelId byte, mapId uint32, referenceCharacterId uint32, o model.Operator[session.Model]) error {
			p := OtherCharacterIdsInMapModelProvider(l)(ctx)(worldId, channelId, mapId, referenceCharacterId)
			return session.ForEachByCharacterId(tenant.MustFromContext(ctx))(p, o)
		}
	}
}
