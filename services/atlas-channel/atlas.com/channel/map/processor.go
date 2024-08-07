package _map

import (
	"atlas-channel/session"
	"atlas-channel/tenant"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func CharacterIdsInMapModelProvider(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32) model.Provider[[]uint32] {
	return func(worldId byte, channelId byte, mapId uint32) model.Provider[[]uint32] {
		return requests.SliceProvider[RestModel, uint32](l)(requestCharactersInMap(l, span, tenant)(worldId, channelId, mapId), Extract)
	}
}

func ForSessionsInMap(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32, o model.Operator[session.Model]) {
	return func(worldId byte, channelId byte, mapId uint32, o model.Operator[session.Model]) {
		session.ForEachByCharacterId(tenant)(CharacterIdsInMapModelProvider(l, span, tenant)(worldId, channelId, mapId), o)
	}
}

func NotCharacterIdFilter(referenceCharacterId uint32) func(characterId uint32) bool {
	return func(characterId uint32) bool {
		return referenceCharacterId != characterId
	}
}

func OtherCharacterIdsInMapModelProvider(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32, referenceCharacterId uint32) model.Provider[[]uint32] {
	return func(worldId byte, channelId byte, mapId uint32, referenceCharacterId uint32) model.Provider[[]uint32] {
		return model.FilteredProvider(CharacterIdsInMapModelProvider(l, span, tenant)(worldId, channelId, mapId), NotCharacterIdFilter(referenceCharacterId))
	}
}

func ForOtherSessionsInMap(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32, referenceCharacterId uint32, o model.Operator[session.Model]) {
	return func(worldId byte, channelId byte, mapId uint32, referenceCharacterId uint32, o model.Operator[session.Model]) {
		p := OtherCharacterIdsInMapModelProvider(l, span, tenant)(worldId, channelId, mapId, referenceCharacterId)
		session.ForEachByCharacterId(tenant)(p, o)
	}
}
