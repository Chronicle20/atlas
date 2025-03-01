package chalkboard

import (
	"atlas-chalkboards/kafka/producer"
	"context"
	"errors"
	"github.com/sirupsen/logrus"
)

func GetById(characterId uint32) (Model, error) {
	m, ok := getRegistry().Get(characterId)
	if !ok {
		return Model{}, errors.New("not found")
	}
	return Model{
		id:      characterId,
		message: m,
	}, nil
}

func Clear(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32) error {
		return func(worldId byte, channelId byte, mapId uint32, characterId uint32) error {
			existed := getRegistry().Clear(characterId)
			if existed {
				l.Debugf("Clearing chalkboard for [%d].", characterId)
				return producer.ProviderImpl(l)(ctx)(EnvEventTopicStatus)(clearStatusEventProvider(worldId, channelId, mapId, characterId))
			}
			return nil
		}
	}
}

func Set(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string) error {
		return func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string) error {
			// TODO ensure they are in a place that this can be set.
			// TODO ensure they are alive.
			l.Debugf("Setting chalkboard to [%s] for [%d].", message, characterId)
			getRegistry().Set(characterId, message)
			return producer.ProviderImpl(l)(ctx)(EnvEventTopicStatus)(setStatusEventProvider(worldId, channelId, mapId, characterId, message))
		}
	}
}
