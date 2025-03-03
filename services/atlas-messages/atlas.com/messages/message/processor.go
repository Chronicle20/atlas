package message

import (
	"atlas-messages/character"
	"atlas-messages/command"
	"atlas-messages/kafka/producer"
	"context"
	"errors"
	"github.com/sirupsen/logrus"
)

func HandleGeneral(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, balloonOnly bool) error {
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

			err = producer.ProviderImpl(l)(ctx)(EnvEventTopicChat)(generalChatEventProvider(worldId, channelId, mapId, characterId, message, balloonOnly))
			if err != nil {
				l.WithError(err).Errorf("Unable to relay message from character [%d].", c.Id())
			}
			return err
		}
	}
}

func HandleMulti(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, chatType string, recipients []uint32) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, chatType string, recipients []uint32) error {
		return func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, chatType string, recipients []uint32) error {
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

			err = producer.ProviderImpl(l)(ctx)(EnvEventTopicChat)(multiChatEventProvider(worldId, channelId, mapId, characterId, message, chatType, recipients))
			if err != nil {
				l.WithError(err).Errorf("Unable to relay message from character [%d].", c.Id())
			}
			return err
		}
	}
}

func HandleWhisper(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, recipientName string) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, recipientName string) error {
		return func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, recipientName string) error {
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

			tc, err := character.GetByName(l)(ctx)(recipientName)
			if err != nil {
				l.WithError(err).Errorf("Unable to locate recipient [%s].", recipientName)
				return err
			}

			if c.WorldId() != tc.WorldId() {
				return errors.New("not in world")
			}

			err = producer.ProviderImpl(l)(ctx)(EnvEventTopicChat)(whisperChatEventProvider(worldId, channelId, mapId, characterId, message, tc.Id()))
			if err != nil {
				l.WithError(err).Errorf("Unable to relay message from character [%d].", c.Id())
			}
			return err
		}
	}
}

func HandleMessenger(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, recipients []uint32) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, recipients []uint32) error {
		return func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, recipients []uint32) error {
			c, err := character.GetById(l)(ctx)(characterId)
			if err != nil {
				l.WithError(err).Errorf("Unable to locate character chatting [%d].", characterId)
				return err
			}

			err = producer.ProviderImpl(l)(ctx)(EnvEventTopicChat)(messengerChatEventProvider(worldId, channelId, mapId, characterId, message, recipients))
			if err != nil {
				l.WithError(err).Errorf("Unable to relay message from character [%d].", c.Id())
			}
			return err
		}
	}
}
