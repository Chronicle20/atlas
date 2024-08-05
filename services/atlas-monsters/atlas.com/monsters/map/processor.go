package _map

import (
	"atlas-monsters/tenant"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func GetCharacterIdsInMap(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32) ([]uint32, error) {
	return func(worldId byte, channelId byte, mapId uint32) ([]uint32, error) {
		return requests.SliceProvider[RestModel, uint32](l)(requestCharactersInMap(l, span, tenant)(worldId, channelId, mapId), Extract)()
	}
}
