package character

import (
	"atlas-portals/tenant"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func EnableActions(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, characterId uint32) {
	return func(worldId byte, channelId byte, characterId uint32) {
		emitEnableActions(l, span, tenant)(worldId, channelId, characterId)
	}
}
