package _map

import (
	"atlas-channel/session"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

func CharacterIdsInMapModelProvider(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32) model.Provider[[]uint32] {
	return func(worldId byte, channelId byte, mapId uint32) model.Provider[[]uint32] {
		return requests.SliceProvider[RestModel, uint32](l)(requestCharactersInMap(ctx, tenant)(worldId, channelId, mapId), Extract)
	}
}

func GetCharacterIdsInMap(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32) ([]uint32, error) {
	return func(worldId byte, channelId byte, mapId uint32) ([]uint32, error) {
		return CharacterIdsInMapModelProvider(l, ctx, tenant)(worldId, channelId, mapId)()
	}
}

func ForSessionsInMap(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32, o model.Operator[session.Model]) error {
	return func(worldId byte, channelId byte, mapId uint32, o model.Operator[session.Model]) error {
		return session.ForEachByCharacterId(tenant)(CharacterIdsInMapModelProvider(l, ctx, tenant)(worldId, channelId, mapId), o)
	}
}

func NotCharacterIdFilter(referenceCharacterId uint32) func(characterId uint32) bool {
	return func(characterId uint32) bool {
		return referenceCharacterId != characterId
	}
}

func OtherCharacterIdsInMapModelProvider(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32, referenceCharacterId uint32) model.Provider[[]uint32] {
	return func(worldId byte, channelId byte, mapId uint32, referenceCharacterId uint32) model.Provider[[]uint32] {
		return model.FilteredProvider(CharacterIdsInMapModelProvider(l, ctx, tenant)(worldId, channelId, mapId), NotCharacterIdFilter(referenceCharacterId))
	}
}

func ForOtherSessionsInMap(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32, referenceCharacterId uint32, o model.Operator[session.Model]) error {
	return func(worldId byte, channelId byte, mapId uint32, referenceCharacterId uint32, o model.Operator[session.Model]) error {
		p := OtherCharacterIdsInMapModelProvider(l, ctx, tenant)(worldId, channelId, mapId, referenceCharacterId)
		return session.ForEachByCharacterId(tenant)(p, o)
	}
}
