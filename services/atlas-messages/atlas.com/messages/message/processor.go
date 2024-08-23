package message

import (
	"atlas-messages/character"
	"atlas-messages/command"
	"atlas-messages/kafka/producer"
	"atlas-messages/tenant"
	"context"
	"github.com/sirupsen/logrus"
)

func Handle(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, balloonOnly bool) error {
	return func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, balloonOnly bool) error {
		c, err := character.GetById(l, ctx, tenant)(characterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to locate character chatting [%d].", characterId)
			return err
		}

		e, found := command.Registry().Get(l, ctx, tenant, worldId, channelId, c, message)
		if found {
			err = e(l, ctx, tenant)
			if err != nil {
				l.WithError(err).Errorf("Unable to execute command for character [%d]. Command=[%s]", c.Id(), message)
			}
			return err
		}

		err = GeneralChat(l, ctx, tenant)(worldId, channelId, mapId, characterId, message, balloonOnly)
		if err != nil {
			l.WithError(err).Errorf("Unable to relay message from character [%d].", c.Id())
		}
		return err
	}
}

func GeneralChat(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, balloonOnly bool) error {
	return func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, balloonOnly bool) error {
		return producer.ProviderImpl(l)(ctx)(EnvEventTopicGeneralChat)(generalChatEventProvider(tenant)(worldId, channelId, mapId, characterId, message, balloonOnly))
	}
}
