package character

import (
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

const consumerStatusEvent = "character_command"

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
