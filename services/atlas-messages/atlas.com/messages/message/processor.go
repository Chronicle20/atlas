package message

import (
	"atlas-messages/character"
	"atlas-messages/command"
	"atlas-messages/kafka/producer"
	"atlas-messages/tenant"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func Handle(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, balloonOnly bool) error {
	return func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, balloonOnly bool) error {
		c, err := character.GetById(l, span, tenant)(characterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to locate character chatting [%d].", characterId)
			return err
		}

		e, found := command.Registry().Get(l, span, tenant, worldId, channelId, c, message)
		if found {
			err = e(l, span, tenant)
			if err != nil {
				l.WithError(err).Errorf("Unable to execute command for character [%d]. Command=[%s]", c.Id(), message)
			}
			return err
		}

		err = GeneralChat(l, span, tenant)(worldId, channelId, mapId, characterId, message, balloonOnly)
		if err != nil {
			l.WithError(err).Errorf("Unable to relay message from character [%d].", c.Id())
		}
		return err
	}
}

func GeneralChat(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, balloonOnly bool) error {
	return func(worldId byte, channelId byte, mapId uint32, characterId uint32, message string, balloonOnly bool) error {
		return producer.ProviderImpl(l)(span)(EnvEventTopicGeneralChat)(generalChatEventProvider(tenant)(worldId, channelId, mapId, characterId, message, balloonOnly))
	}
}
