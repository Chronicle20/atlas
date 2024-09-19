package _map

import (
	"atlas-maps/kafka/producer"
	"atlas-maps/map/character"
	"context"
	"github.com/sirupsen/logrus"
)

func Transition(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, oldMapId uint32) {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, oldMapId uint32) {
		return func(worldId byte, channelId byte, mapId uint32, characterId uint32, oldMapId uint32) {
			Exit(l)(ctx)(worldId, channelId, oldMapId, characterId)
			Enter(l)(ctx)(worldId, channelId, mapId, characterId)
		}
	}
}

func Enter(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32) {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32) {
		return func(worldId byte, channelId byte, mapId uint32, characterId uint32) {
			character.Enter(ctx)(worldId, channelId, mapId, characterId)
			_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicMapStatus)(enterMapProvider(worldId, channelId, mapId, characterId))
		}
	}
}

func Exit(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32) {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32) {
		return func(worldId byte, channelId byte, mapId uint32, characterId uint32) {
			character.Exit(ctx)(worldId, channelId, mapId, characterId)
			_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicMapStatus)(exitMapProvider(worldId, channelId, mapId, characterId))
		}
	}
}
