package character

import (
	"atlas-channel/character"
	"atlas-channel/kafka"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"atlas-channel/tenant"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

const consumerStatusEvent = "character_status"

func StatusEventConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return kafka.NewConfig(l)(consumerStatusEvent)(EnvEventTopicCharacterStatus)(groupId)
	}
}

func StatusEventStatChangedRegister(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger) (string, handler.Handler) {
	return func(l logrus.FieldLogger) (string, handler.Handler) {
		return kafka.LookupTopic(l)(EnvEventTopicCharacterStatus), message.AdaptHandler(message.PersistentConfig(handleStatusEventStatChanged(sc, wp)))
	}
}

func StatusEventMapChangedRegister(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger) (string, handler.Handler) {
	return func(l logrus.FieldLogger) (string, handler.Handler) {
		return kafka.LookupTopic(l)(EnvEventTopicCharacterStatus), message.AdaptHandler(message.PersistentConfig(handleStatusEventMapChanged(sc, wp)))
	}
}

func handleStatusEventStatChanged(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, span opentracing.Span, event statusEvent[statusEventStatChangedBody]) {
	return func(l logrus.FieldLogger, span opentracing.Span, event statusEvent[statusEventStatChangedBody]) {
		if !sc.Is(event.Tenant, event.WorldId, event.Body.ChannelId) {
			return
		}

		if event.Type != EventCharacterStatusTypeStatChanged {
			return
		}

		session.IfPresentByCharacterId(event.Tenant)(event.CharacterId, statChanged(l, span, event.Tenant, wp)(event.Body.ExclRequestSent))
	}
}

func statChanged(l logrus.FieldLogger, _ opentracing.Span, _ tenant.Model, wp writer.Producer) func(exclRequestSent bool) model.Operator[session.Model] {
	statChangedFunc := session.Announce(l)(wp)(writer.StatChanged)
	return func(exclRequestSent bool) model.Operator[session.Model] {
		return func(s session.Model) error {
			err := statChangedFunc(s, writer.StatChangedBody(l)(exclRequestSent))
			if err != nil {
				l.WithError(err).Errorf("Unable to write [%s] for character [%d].", writer.StatChanged, s.CharacterId())
				return err
			}
			return nil
		}
	}
}

func handleStatusEventMapChanged(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, span opentracing.Span, event statusEvent[statusEventMapChangedBody]) {
	return func(l logrus.FieldLogger, span opentracing.Span, event statusEvent[statusEventMapChangedBody]) {
		if !sc.Is(event.Tenant, event.WorldId, event.Body.ChannelId) {
			return
		}

		if event.Type != EventCharacterStatusTypeMapChanged {
			return
		}

		session.IfPresentByCharacterId(event.Tenant)(event.CharacterId, warpCharacter(l, span, event.Tenant, wp)(event))
	}
}

func warpCharacter(l logrus.FieldLogger, span opentracing.Span, t tenant.Model, wp writer.Producer) func(event statusEvent[statusEventMapChangedBody]) model.Operator[session.Model] {
	setFieldFunc := session.Announce(l)(wp)(writer.SetField)
	return func(event statusEvent[statusEventMapChangedBody]) model.Operator[session.Model] {
		return func(s session.Model) error {
			c, err := character.GetById(l, span, t)(s.CharacterId())
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve character [%d].", s.CharacterId())
				return err
			}

			err = setFieldFunc(s, writer.WarpToMapBody(l, s.Tenant())(s.ChannelId(), event.Body.TargetMapId, event.Body.TargetPortalId, c.Hp()))
			if err != nil {
				l.WithError(err).Errorf("Unable to show set field response for character [%d]", c.Id())
				return err
			}
			return nil
		}
	}
}
