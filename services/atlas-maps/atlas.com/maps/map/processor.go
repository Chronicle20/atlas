package _map

import (
	"atlas-maps/kafka/producer"
	"atlas-maps/map/character"
	"atlas-maps/tenant"
	"context"
	"github.com/sirupsen/logrus"
)

func Transition(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32, characterId uint32, oldMapId uint32) {
	return func(worldId byte, channelId byte, mapId uint32, characterId uint32, oldMapId uint32) {
		Exit(l, ctx, tenant)(worldId, channelId, oldMapId, characterId)
		Enter(l, ctx, tenant)(worldId, channelId, mapId, characterId)
	}
}

func Enter(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32, characterId uint32) {
	return func(worldId byte, channelId byte, mapId uint32, characterId uint32) {
		character.Enter(tenant)(worldId, channelId, mapId, characterId)
		_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicMapStatus)(enterMapProvider(tenant, worldId, channelId, mapId, characterId))
	}
}

func Exit(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32, characterId uint32) {
	return func(worldId byte, channelId byte, mapId uint32, characterId uint32) {
		character.Exit(tenant)(worldId, channelId, mapId, characterId)
		_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicMapStatus)(exitMapProvider(tenant, worldId, channelId, mapId, characterId))
	}
}
