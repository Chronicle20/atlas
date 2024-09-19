package message

import (
	"atlas-messages/character"
	"atlas-messages/command"
	"atlas-messages/kafka/producer"
	"context"
	"github.com/sirupsen/logrus"
)

func Handle(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, balloonOnly bool) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, balloonOnly bool) error {
		return func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, balloonOnly bool) error {
			c, err := character.GetById(l)(ctx)(characterId)
			if err != nil {
				l.WithError(err).Errorf("Unable to locate character chatting [%d].", characterId)
				return err
			}

			e, found := command.Registry().Get(l, ctx, worldId, channelId, c, message)
			if found {
				err = e(l)(ctx)
				if err != nil {
					l.WithError(err).Errorf("Unable to execute command for character [%d]. Command=[%s]", c.Id(), message)
				}
				return err
			}

			err = GeneralChat(l)(ctx)(worldId, channelId, mapId, characterId, message, balloonOnly)
			if err != nil {
				l.WithError(err).Errorf("Unable to relay message from character [%d].", c.Id())
			}
			return err
		}
	}
}

func GeneralChat(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, balloonOnly bool) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, balloonOnly bool) error {
		return func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, balloonOnly bool) error {
			return producer.ProviderImpl(l)(ctx)(EnvEventTopicGeneralChat)(generalChatEventProvider(worldId, channelId, mapId, characterId, message, balloonOnly))
		}
	}
}
