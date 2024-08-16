package _map

import (
	"atlas-monsters/tenant"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func CharacterIdsInMapProvider(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32) model.Provider[[]uint32] {
	return func(worldId byte, channelId byte, mapId uint32) model.Provider[[]uint32] {
		return requests.SliceProvider[RestModel, uint32](l)(requestCharactersInMap(l, span, tenant)(worldId, channelId, mapId), Extract)
	}
}

func GetCharacterIdsInMap(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32) ([]uint32, error) {
	return func(worldId byte, channelId byte, mapId uint32) ([]uint32, error) {
		return CharacterIdsInMapProvider(l, span, tenant)(worldId, channelId, mapId)()
	}
}
