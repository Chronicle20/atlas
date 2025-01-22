package guild

import (
	"atlas-channel/kafka/producer"
	"context"
	"github.com/sirupsen/logrus"
)

func RequestCreate(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, name string) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, name string) error {
		return func(worldId byte, channelId byte, mapId uint32, characterId uint32, name string) error {
			l.Debugf("Character [%d] attempting to create guild [%s] in world [%d] channel [%d] map [%d].", characterId, name, worldId, channelId, mapId)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(requestCreateProvider(worldId, channelId, mapId, characterId, name))
		}
	}
}

func CreationAgreement(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, agreed bool) error {
	return func(ctx context.Context) func(characterId uint32, agreed bool) error {
		return func(characterId uint32, agreed bool) error {
			l.Debugf("Character [%d] responded to guild creation agreement with [%t].", characterId, agreed)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(creationAgreementProvider(characterId, agreed))
		}
	}
}
