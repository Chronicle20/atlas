package seed

import (
	"atlas-login/character"
	consumer2 "atlas-login/kafka/consumer"
	"atlas-login/kafka/message/seed"
	"atlas-login/session"
	"atlas-login/socket/writer"
	"context"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("seed_status_event")(seed.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(ten tenant.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(ten tenant.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
			return func(rf func(topic string, handler handler.Handler) (string, error)) error {
				t, _ := topic.EnvProvider(l)(seed.EnvEventTopicStatus)()
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCreatedStatusEvent(ten, wp)))); err != nil {
					return err
				}
				return nil
			}
		}
	}
}

func handleCreatedStatusEvent(t tenant.Model, wp writer.Producer) message.Handler[seed.StatusEvent[seed.CreatedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e seed.StatusEvent[seed.CreatedStatusEventBody]) {
		if e.Type != seed.StatusEventTypeCreated {
			return
		}

		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		_ = session.NewProcessor(l, ctx).IfPresentByAccountId(e.AccountId, func(s session.Model) error {
			cp := character.NewProcessor(l, ctx)
			c, err := cp.GetById(cp.InventoryDecorator())(e.Body.CharacterId)
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve newly created character [%d] for account [%d].", e.Body.CharacterId, e.AccountId)
				err = session.Announce(l)(wp)(writer.AddCharacterEntry)(s, writer.AddCharacterErrorBody(l, t)(writer.AddCharacterCodeUnknownError))
				if err != nil {
					l.WithError(err).Errorf("Unable to show character creation error.")
				}
				return err
			}
			err = session.Announce(l)(wp)(writer.AddCharacterEntry)(s, writer.AddCharacterEntryBody(l, t)(c))
			if err != nil {
				l.WithError(err).Errorf("Unable to show newly created character.")
			}
			return nil
		})
	}
}
