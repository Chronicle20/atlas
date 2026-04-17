package seed

import (
	"atlas-login/character"
	consumer2 "atlas-login/kafka/consumer"
	"atlas-login/kafka/message/seed"
	"atlas-login/session"
	"atlas-login/socket/writer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleFailedStatusEvent(ten, wp)))); err != nil {
					return err
				}
				return nil
			}
		}
	}
}

// handleFailedStatusEvent writes AddCharacterCodeUnknownError to the waiting
// session when the orchestrator's saga fails (PRD §4.5 / plan Phase 8). The
// factory bridge (Phase 7) re-emits orchestrator FAILED as seed FAILED with
// the account-level envelope `AccountId` set; we resolve the session by that
// id. If the session is already disconnected, we log at INFO and drop — no
// panic. Reason is carried through for server-side logs only; the client
// receives a generic error code.
func handleFailedStatusEvent(t tenant.Model, wp writer.Producer) message.Handler[seed.StatusEvent[seed.FailedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e seed.StatusEvent[seed.FailedStatusEventBody]) {
		if e.Type != seed.StatusEventTypeFailed {
			return
		}
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}
		if e.AccountId == 0 {
			l.WithField("reason", e.Body.Reason).Warn("Seed FAILED with accountId=0; cannot resolve session.")
			return
		}
		found := false
		_ = session.NewProcessor(l, ctx).IfPresentByAccountId(e.AccountId, func(s session.Model) error {
			found = true
			l.WithFields(logrus.Fields{
				"account_id": e.AccountId,
				"reason":     e.Body.Reason,
			}).Info("Character creation failed; announcing error to session.")
			if err := session.Announce(l)(ctx)(wp)(charpkt.AddCharacterEntryWriter)(writer.AddCharacterErrorBody(writer.AddCharacterCodeUnknownError))(s); err != nil {
				l.WithError(err).WithField("account_id", e.AccountId).Error("Unable to announce character creation failure to session.")
			}
			return nil
		})
		if !found {
			l.WithFields(logrus.Fields{
				"account_id": e.AccountId,
				"reason":     e.Body.Reason,
			}).Info("Seed FAILED received for disconnected session; dropping.")
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
				err = session.Announce(l)(ctx)(wp)(charpkt.AddCharacterEntryWriter)(writer.AddCharacterErrorBody(writer.AddCharacterCodeUnknownError))(s)
				if err != nil {
					l.WithError(err).Errorf("Unable to show character creation error.")
				}
				return err
			}
			err = session.Announce(l)(ctx)(wp)(charpkt.AddCharacterEntryWriter)(writer.AddCharacterEntryBody(c))(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to show newly created character.")
			}
			return nil
		})
	}
}
