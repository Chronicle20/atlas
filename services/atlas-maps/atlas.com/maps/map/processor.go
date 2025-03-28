package _map

import (
	"atlas-maps/kafka/producer"
	"atlas-maps/map/character"
	monster2 "atlas-maps/map/monster"
	"atlas-maps/reactor"
	"context"
	"github.com/sirupsen/logrus"
)

func Enter(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32) {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32) {
		return func(worldId byte, channelId byte, mapId uint32, characterId uint32) {
			character.Enter(ctx)(worldId, channelId, mapId, characterId)

			go func() {
				_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicMapStatus)(enterMapProvider(worldId, channelId, mapId, characterId))
			}()
			go monster2.Spawn(l)(ctx)(worldId, channelId, mapId)
			go reactor.Spawn(l)(ctx)(worldId, channelId, mapId)
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

func TransitionMap(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, oldMapId uint32) {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, oldMapId uint32) {
		return func(worldId byte, channelId byte, mapId uint32, characterId uint32, oldMapId uint32) {
			Exit(l)(ctx)(worldId, channelId, oldMapId, characterId)
			Enter(l)(ctx)(worldId, channelId, mapId, characterId)
		}
	}
}

func TransitionChannel(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, oldChannelId byte, characterId uint32, mapId uint32) {
	return func(ctx context.Context) func(worldId byte, channelId byte, oldChannelId byte, characterId uint32, mapId uint32) {
		return func(worldId byte, channelId byte, oldChannelId byte, characterId uint32, mapId uint32) {
			Exit(l)(ctx)(worldId, oldChannelId, mapId, characterId)
			Enter(l)(ctx)(worldId, channelId, mapId, characterId)
		}
	}
}
