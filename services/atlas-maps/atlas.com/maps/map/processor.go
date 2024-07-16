package _map

import (
	"atlas-maps/tenant"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func Transition(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32, characterId uint32, oldMapId uint32) {
	return func(worldId byte, channelId byte, mapId uint32, characterId uint32, oldMapId uint32) {
		Exit(l, span, tenant)(worldId, channelId, oldMapId, characterId)
		Enter(l, span, tenant)(worldId, channelId, mapId, characterId)
	}
}

func Enter(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32, characterId uint32) {
	return func(worldId byte, channelId byte, mapId uint32, characterId uint32) {
		//character.GetRegistry().AddToMap(worldId, channelId, mapId, characterId)
		emitEnterMap(l, span, tenant)(worldId, channelId, mapId, characterId)
	}
}

func Exit(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32, characterId uint32) {
	return func(worldId byte, channelId byte, mapId uint32, characterId uint32) {
		//mk, err := character.GetRegistry().GetMapId(characterId)
		//if err == nil {
		//	character.GetRegistry().RemoveFromMap(characterId)
		emitExitMap(l, span, tenant)(worldId, channelId, mapId, characterId)
		//}
	}
}
