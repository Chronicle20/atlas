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
